package archive

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/grafana/github-repo-metrics/pkg/common"
	"github.com/grafana/grafana/pkg/components/simplejson"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const maxGoRoutines = 6
const maxGoProcess = 8

var eventCount int64

func (ad *ArchiveDownloader) buildUrlsDownload(archFiles []*common.ArchiveFile, st, stopDate time.Time) []*common.ArchiveFile {
	var result []*common.ArchiveFile

	// create lookup index based in ArchiveFile ID
	index := map[int64]*common.ArchiveFile{}
	for _, a := range archFiles {
		index[a.ID] = a
	}

	for st.Unix() < stopDate.Unix() {
		archivedFile := common.NewArchiveFile(st.Year(), int(st.Month()), st.Day(), st.Hour())
		_, exist := index[archivedFile.ID]
		if !exist {
			result = append(result, archivedFile)
		}

		st = st.Add(time.Hour)
	}

	return result
}

type ArchiveDownloader struct {
	engine    *xorm.Engine
	url       string
	orgNames  []string
	startDate time.Time
	stopDate  time.Time
	doneChan  chan time.Time
	eventChan chan *common.GithubEvent
}

func NewArchiveDownloader(engine *xorm.Engine, url string, orgNames []string, startDate, stopDate time.Time, doneChan chan time.Time) *ArchiveDownloader {
	return &ArchiveDownloader{
		engine:    engine,
		url:       url,
		orgNames:  orgNames,
		startDate: startDate,
		stopDate:  stopDate,
		doneChan:  doneChan,
		eventChan: make(chan *common.GithubEvent, 10),
	}
}

func (ad *ArchiveDownloader) spawnWorker(index int, wg *sync.WaitGroup, downloadUrls chan *common.ArchiveFile, done chan bool) {
	wg.Add(1)
	go func(workerID int) {
		defer wg.Done()

		log.Printf("starting workerID #%d\n", workerID)

		for {
			select {
			case <-done:
				log.Printf("worker #%d is complete\n", workerID)
				return
			case u := <-downloadUrls:
				err := ad.download(u)
				if err != nil {
					log.Printf("failed to download file. createdAt: %v error: %+v\n", u.CreatedAt, err)
				}
			}
		}
	}(index)
}

func (ad *ArchiveDownloader) spawnDatabaseWriter(wg *sync.WaitGroup, eventChan chan *common.GithubEvent, done chan bool) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			case event := <-eventChan:
				err := ad.insertIntoDatabase(event)
				if err != nil {
					log.Fatalf("failed to delete github event. error: %+v", err)
				}
			}
		}
	}()
}

// DownloadEvents start to download all archive events
func (ad *ArchiveDownloader) DownloadEvents() error {
	start := time.Now()

	var archFiles []*common.ArchiveFile
	err := ad.engine.Find(&archFiles)
	if err != nil {
		return err
	}

	log.Printf("found %v arch files", len(archFiles))

	urls := ad.buildUrlsDownload(archFiles, ad.startDate, ad.stopDate)
	var downloadUrls = make(chan *common.ArchiveFile)
	var done = make(chan bool)
	wg := sync.WaitGroup{}

	// start workers
	for i := 0; i < maxGoRoutines; i++ {
		ad.spawnWorker(i, &wg, downloadUrls, done)
	}

	ad.spawnDatabaseWriter(&wg, ad.eventChan, done)

	go func() {
		i := 0

		for {
			select {
			case <-ad.doneChan:
				log.Println("closing down gracefully. cancelled by parent")
				close(done)
				return
			default:
				if i == len(urls) {
					close(done)
					return
				}
				downloadUrls <- urls[i]
				i++
			}
		}
	}()

	defer close(downloadUrls)

	// wait for all workers to complete
	wg.Wait()

	log.Printf("filtered event: %d - elapsed: %v\n", eventCount, time.Since(start))

	return nil
}

func (ad *ArchiveDownloader) spawnLineProcessor(file *common.ArchiveFile, wg *sync.WaitGroup, lines chan string) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for line := range lines {
			ge := common.GithubEventJSON{}
			err := json.Unmarshal([]byte(line), &ge)
			if err != nil {
				log.Printf("failed to parse json. createdAt: %v err %+v\n", file.CreatedAt, err)
				return
			}

			for _, v := range ad.orgNames {
				if ge.Org != nil && ge.Org.Login == v {
					id, _ := strconv.ParseInt(ge.ID, 10, 0)
					fullJSON, _ := simplejson.NewJson([]byte(line))
					ad.eventChan <- &common.GithubEvent{ID: id, CreatedAt: ge.CreatedAt, Data: fullJSON}
					eventCount++
				}
			}
		}
	}()
}

func (ad *ArchiveDownloader) buildDownloadURL(file *common.ArchiveFile) string {
	ft := time.Unix(file.ID, 0).UTC()
	return fmt.Sprintf(ad.url, ft.Year(), ft.Month(), ft.Day(), ft.Hour())
}

func (ad *ArchiveDownloader) download(file *common.ArchiveFile) error {
	url := ad.buildDownloadURL(file)
	res, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to download json file")
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)
	defer res.Body.Close()

	zr, err := gzip.NewReader(buf)
	if err != nil {
		return errors.Wrap(err, "parsing compress content")
	}
	zr.Multistream(false)
	bites, err := ioutil.ReadAll(zr)
	if err != nil {
		return err
	}

	res.Body.Close()

	rrr := bytes.NewReader(bites)

	lines := make(chan string)
	wg := sync.WaitGroup{}

	//spawn go routines that will marshal strings to json
	for i := 0; i <= maxGoProcess; i++ {
		ad.spawnLineProcessor(file, &wg, lines)
	}

	// decompress response body and send to workers.
	for {

		r := bufio.NewReaderSize(rrr, 2048*2048)

		line, isPrefix, err := r.ReadLine()
		var s string
		for err == nil {
			if isPrefix {
				s += string(line)
			} else {
				s = string(line)
			}

			if !isPrefix {
				lines <- string(line)
			}

			line, isPrefix, err = r.ReadLine()
		}

		if err != io.EOF {
			log.Printf("failed to read line. createdAt: %v error:%v\n", file.CreatedAt, err)
			break
		}

		if err == io.EOF {
			break
		}
	}

	close(lines)

	wg.Wait()

	_, err = ad.engine.Insert(file)
	if err != nil {
		return err
	}

	return zr.Close()
}

func (ad *ArchiveDownloader) insertIntoDatabase(event *common.GithubEvent) error {
	//remove the event first to make development easier.
	_, err := ad.engine.Exec("DELETE FROM github_event WHERE ID = ? ", event.ID)
	if err != nil {
		return err
	}

	_, err = ad.engine.Insert(event)
	return err
}
