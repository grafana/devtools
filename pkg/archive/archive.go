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
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/grafana/github-repo-metrics/pkg/common"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const maxGoRoutines = 6
const maxGoProcess = 8

var eventCount int64
var filesWithErrors = []string{}

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
		if !exist || ad.overrideAllFiles {
			result = append(result, archivedFile)
		}

		st = st.Add(time.Hour)
	}

	return result
}

// ArchiveDownloader downloads, decompress and store events for github orgs
type ArchiveDownloader struct {
	engine           *xorm.Engine
	url              string
	orgNames         []string
	startDate        time.Time
	stopDate         time.Time
	doneChan         chan time.Time
	eventChan        chan *common.GithubEvent
	overrideAllFiles bool
}

// NewArchiveDownloader creates a new downloader
func NewArchiveDownloader(engine *xorm.Engine, overrideAllFiles bool, url string, orgNames []string, startDate, stopDate time.Time, doneChan chan time.Time) *ArchiveDownloader {
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
					filesWithErrors = append(filesWithErrors, fmt.Sprintf("%v", u.CreatedAt))
					log.Printf("failed to download file. createdAt: %v error: %+v\n", u.CreatedAt, err)
				}
			}
		}
	}(index)
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

	log.Printf("filtered event: %d - failed: %v - elapsed: %v\n", eventCount, len(filesWithErrors), time.Since(start))
	log.Printf("dates with failed downloads: %v", strings.Join(filesWithErrors, ","))
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
					ad.insertIntoDatabase(&common.GithubEvent{ID: id, CreatedAt: ge.CreatedAt, Data: line})
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
	log.Printf("downloading file for : %v\n", file.CreatedAt)
	url := ad.buildDownloadURL(file)
	res, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to download json file")
	}

	if res.StatusCode != 200 {
		log.Printf("bad http status code for file: %v status: %v\n", file.CreatedAt, res.StatusCode)
		return nil
	}

	//create reader that can reader gziped contetn
	zipReader, err := gzip.NewReader(res.Body)
	if err != nil {
		return errors.Wrap(err, "parsing compress content")
	}

	//read all zipped data into memory.
	byteArray, err := ioutil.ReadAll(zipReader)
	if err != nil {
		return err
	}

	defer zipReader.Close()
	defer res.Body.Close()

	inMemReader := bytes.NewReader(byteArray)

	lines := make(chan string)
	wg := sync.WaitGroup{}

	//spawn go routines that will marshal strings to json
	for i := 0; i <= maxGoProcess; i++ {
		ad.spawnLineProcessor(file, &wg, lines)
	}

	// decompress response body and send to workers.
	bufferReader := bufio.NewReaderSize(inMemReader, 2048*2048)
	for {
		line, isPrefix, err := bufferReader.ReadLine()
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

			line, isPrefix, err = bufferReader.ReadLine()
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

	return nil
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
