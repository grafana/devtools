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

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var allEvents []GithubEventJson
var lock sync.Mutex

const max_go_routines = 4
const max_go_process = 6

var dlTotal int64
var dlCount int64
var pTotal int64
var pCount int64

func (ad *ArchiveDownloader) buildUrlsDownload(archFiles []*ArchiveFile, st, stopDate time.Time) []*ArchiveFile {
	var result []*ArchiveFile

	index := map[int64]*ArchiveFile{}

	// create lookup index based in ArchiveFile ID
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
	engine  *xorm.Engine
	url     string
	repoIds []int64
}

func NewArchiveDownloader(engine *xorm.Engine, url string, repoIds []int64) *ArchiveDownloader {
	return &ArchiveDownloader{
		engine:  engine,
		url:     url,
		repoIds: repoIds,
	}
}

func (ad *ArchiveDownloader) DownloadEvents() {
	var downloadUrls = make(chan *ArchiveFile, 0)
	start := time.Now()

	var archFiles []*ArchiveFile
	err := ad.engine.Find(&archFiles)
	if err != nil {
		log.Fatalf("could not find archivefile. error: %v", err)
	}

	log.Printf("found %v arch files", len(archFiles))

	downloadGroup := errgroup.Group{}

	for i := 0; i <= max_go_routines; i++ {
		downloadGroup.Go(
			func() error {
				for u := range downloadUrls {
					err := ad.download(u)
					if err != nil {
						log.Printf("error: %+v failed to download file. error: %v\n", u, err)
					}
				}
				return nil
			})
	}

	// First public event for Grafana was at 16th Jan 2014
	// startDate := time.Date(2014, time.Month(1), 1, 0, 0, 0, 0, time.UTC)

	startDate := time.Date(2015, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
	// stopDate := time.Now()
	// stopDate := time.Date(2018 time.Month(1), 2, 1, 0, 0, 0, time.UTC)
	stopDate := time.Date(2015, time.Month(1), 4, 0, 0, 0, 0, time.UTC)

	urls := ad.buildUrlsDownload(archFiles, startDate, stopDate)
	for _, u := range urls {
		downloadUrls <- u
	}
	close(downloadUrls)

	err = downloadGroup.Wait()
	if err != nil {
		log.Fatalf("error: %+v", err)
	}

	log.Println("filtered event: ", len(allEvents))
	log.Println("elapsed: ", time.Since(start))
	if dlCount > 0 && pCount > 0 {
		log.Println("avg download :", time.Duration(dlTotal/dlCount).String())
		log.Println("avg process  :", time.Duration(pTotal/pCount).String())
	}
}

func (ad *ArchiveDownloader) download(file *ArchiveFile) error {
	start := time.Now()
	ft := time.Unix(file.ID, 0).UTC()
	url := fmt.Sprintf(ad.url, ft.Year(), ft.Month(), ft.Day(), ft.Hour())

	log.Println("downloading url: ", url)

	res, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to download json file")
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)
	defer res.Body.Close()

	dlTotal += int64(time.Now().Sub(start))
	dlCount += 1

	pStart := time.Now()

	zr, err := gzip.NewReader(buf)
	if err != nil {
		return errors.Wrap(err, "parsing compress content")
	}

	var events []GithubEventJson

	lines := make(chan []byte, 0)
	eg := errgroup.Group{}
	for i := 0; i <= max_go_process; i++ {
		eg.Go(func() error {
			for line := range lines {
				ge := GithubEventJson{}
				err := json.Unmarshal(line, &ge)
				if err != nil {
					return errors.Wrap(err, "parsing json")
				}

				for _, v := range ad.repoIds {
					if ge.Repo.ID == v {
						events = append(events, ge)
					}
				}
			}
			return nil
		})
	}

	for {
		zr.Multistream(false)

		scanner := bufio.NewScanner(zr)
		scanner.Buffer([]byte(""), 2048*2048) //increase buffer limit

		for scanner.Scan() {
			lines <- []byte(scanner.Text()) //why does this work? :/
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

	var dbEvents []*GithubEvent
	for _, e := range events {
		dbEvents = append(dbEvents, e.CreateGithubEvent())
	}

	err = ad.insertIntoDatabase(dbEvents)
	if err != nil {
		log.Fatalf("failed to connect to database. error %v", err)
	}

	ad.engine.Insert(file)

	pTotal += int64(time.Now().Sub(pStart))
	pCount += 1

	lock.Lock()
	allEvents = append(allEvents, events...)
	lock.Unlock()
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
