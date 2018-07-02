package archive

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
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
	"golang.org/x/sync/errgroup"
)

var lock sync.Mutex

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

func (ad *ArchiveDownloader) DownloadEvents() error {
	start := time.Now()

	var archFiles []*ArchiveFile
	err := ad.engine.Find(&archFiles)
	if err != nil {
		return err
	}

	log.Printf("found %v arch files", len(archFiles))

	downloadGroup, _ := errgroup.WithContext(context.Background())

	urls := ad.buildUrlsDownload(archFiles, ad.startDate, ad.stopDate)
	var downloadUrls = make(chan *ArchiveFile, len(urls))

	// start workers
	for i := 0; i < maxGoRoutines; i++ {
		workerID := i

		downloadGroup.Go(
			func() error {
				log.Printf("starting workerID #%d\n", workerID)
				for {
					select {
					case <-ad.doneChan:
						log.Printf("closing workerID #%d\n", workerID)
						return nil
					case u := <-downloadUrls:
						if u != nil {
							err := ad.download(u)
							if err != nil {
								log.Printf("error: %+v failed to download file. error: %v\n", u, err)
							}
						} else {
							log.Printf("worker #%d is complete\n", workerID)
							return nil
						}
					}
				}
			})
	}

	// queue up work
	for _, u := range urls {
		downloadUrls <- u
	}

	// send nil to close each worker
	for i := 0; i < maxGoRoutines; i++ {
		downloadUrls <- nil
	}

	// wait for all workers to complete
	err = downloadGroup.Wait()
	if err != nil {
		return err
	}

	log.Printf("filtered event: %d - elapsed: %v\n", eventCount, time.Since(start))

	return nil
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

	lines := make(chan []byte, 0)
	eg := errgroup.Group{}
	for i := 0; i <= maxGoProcess; i++ {
		eg.Go(func() error {
			for line := range lines {
				ge := GithubEventJson{}
				err := json.Unmarshal(line, &ge)
				if err != nil {
					return errors.Wrap(err, "parsing json")
				}

				for _, v := range ad.repoIds {
					if ge.Repo.ID == v {
						ad.insertIntoDatabase([]*GithubEvent{ge.CreateGithubEvent()})
						eventCount++
					}
				}
			}
			return nil
		})
	}

	for {
		zr.Multistream(false)

		scanner := bufio.NewScanner(zr)
		buff := make([]byte, 2048*2048)
		scanner.Buffer(buff, 2048*2048) //increase buffer limit

		for scanner.Scan() {
			lines <- scanner.Bytes()
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

	if eg.Wait() != nil {
		return err
	}

	ad.engine.Insert(file)

	return zr.Close()
}

func (ad *ArchiveDownloader) insertIntoDatabase(events []*GithubEvent) error {

	// we could batch this if we need to write things faster.
	for _, e := range events {
		_, err := ad.engine.Exec("DELETE FROM github_event WHERE ID = ? ", e.ID)
		if err != nil {
			return err
		}

		_, err = ad.engine.Insert(e)
		if err != nil {
			return err
		}
	}

	return nil
}
