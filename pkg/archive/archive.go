package archive

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-xorm/xorm"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const maxGoRoutines = 4
const maxGoProcess = 6

var eventCount int64

func (ad *ArchiveDownloader) buildUrlsDownload(archFiles []*ArchiveFile, st, stopDate time.Time) []*ArchiveFile {
	var result []*ArchiveFile

	// create lookup index based in ArchiveFile ID
	index := map[int64]*ArchiveFile{}
	for _, a := range archFiles {
		index[a.ID] = a
	}

	for st.Unix() < stopDate.Unix() {
		archivedFile := NewArchiveFile(st.Year(), int(st.Month()), st.Day(), st.Hour())
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
	repoIds   []int64
	startDate time.Time
	stopDate  time.Time
	doneChan  chan time.Time
}

func NewArchiveDownloader(engine *xorm.Engine, url string, repoIds []int64, startDate, stopDate time.Time, doneChan chan time.Time) *ArchiveDownloader {
	return &ArchiveDownloader{
		engine:    engine,
		url:       url,
		repoIds:   repoIds,
		startDate: startDate,
		stopDate:  stopDate,
		doneChan:  doneChan,
	}
}

func (ad *ArchiveDownloader) spawnWorker(index int, wg *sync.WaitGroup, downloadUrls chan *ArchiveFile, done chan bool) {
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
					log.Printf("failed to download file. error: %+v\n", err)
				}
			}
		}
	}(index)
}

func (ad *ArchiveDownloader) DownloadEvents() error {
	start := time.Now()

	var archFiles []*ArchiveFile
	err := ad.engine.Find(&archFiles)
	if err != nil {
		return err
	}

	log.Printf("found %v arch files", len(archFiles))

	urls := ad.buildUrlsDownload(archFiles, ad.startDate, ad.stopDate)
	var downloadUrls = make(chan *ArchiveFile)
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

	log.Printf("filtered event: %d - elapsed: %v\n", eventCount, time.Since(start))

	return nil
}

func (ad *ArchiveDownloader) spawnLineProcessor(index int, wg *sync.WaitGroup, lines chan string) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for line := range lines {
			ge := GithubEventJSON{}
			err := json.Unmarshal([]byte(line), &ge)
			if err != nil {
				log.Printf("failed to parse json. err %+v\n", err)
				return
			}

			for _, v := range ad.repoIds {
				if ge.Repo.ID == v {
					ad.insertIntoDatabase(ge.CreateGithubEvent())
					eventCount++
				}
			}
		}
	}()
}

func (ad *ArchiveDownloader) download(file *ArchiveFile) error {
	ft := time.Unix(file.ID, 0).UTC()
	url := fmt.Sprintf(ad.url, ft.Year(), ft.Month(), ft.Day(), ft.Hour())

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

	lines := make(chan string)
	wg := sync.WaitGroup{}
	for i := 0; i <= maxGoProcess; i++ {
		ad.spawnLineProcessor(i, &wg, lines)
	}

	for {
		zr.Multistream(false)

		scanner := bufio.NewScanner(zr)
		buff := make([]byte, 2048*2048)
		scanner.Buffer(buff, 2048*2048) //increase buffer limit

		for scanner.Scan() {
			lines <- scanner.Text()
		}

		err := scanner.Err()
		if err != nil {
			log.Println("reading standard input:", err)
			break
		}

		if scanner.Err() == io.EOF {
			log.Println("scanner EOF")
			break
		}

		err = zr.Reset(buf)
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

func (ad *ArchiveDownloader) insertIntoDatabase(event *GithubEvent) error {
	//remove the event first to make development easier.
	_, err := ad.engine.Exec("DELETE FROM github_event WHERE ID = ? ", event.ID)
	if err != nil {
		return err
	}

	_, err = ad.engine.Insert(event)
	return err
}
