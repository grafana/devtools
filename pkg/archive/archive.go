package main

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

func (ad *ArchiveDownloader) buildUrlsDownload(archFiles []*ArchiveFile, startDate, stopDate time.Time) []*ArchiveFile {
	var result []*ArchiveFile

	for startDate.Unix() < stopDate.Unix() {

		// TODO: use hashset to see if the file have been
		// processed before. This looks like shiet
		archivedFile := &ArchiveFile{
			Year:  startDate.Year(),
			Month: int(startDate.Month()),
			Day:   startDate.Day(),
			Hour:  startDate.Hour(),
		}

		for _, a := range archFiles {
			if archivedFile.Equals(a) {
				break
			}
		}

		result = append(result, archivedFile)
		startDate = startDate.Add(time.Hour)
	}

	return result
}

type ArchiveDownloader struct {
	engine *xorm.Engine
}

func NewArchiveDownloader(engine *xorm.Engine) *ArchiveDownloader {
	return &ArchiveDownloader{
		engine: engine,
	}
}

func (ad *ArchiveDownloader) downloadEvents() {
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

	startDate := time.Date(2015, time.Month(1), 1, 0, 0, 0, 0, time.Local)
	stopDate := time.Now()
	//stopDate := time.Date(2018, time.Month(1), 3, 12, 0, 0, 0, time.Local)
	//stopDate := time.Date(2015, time.Month(1), 2, 0, 0, 0, 0, time.Local)

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
	log.Println("avg download :", time.Duration(dlTotal/dlCount).String())
	log.Println("avg process  :", time.Duration(pTotal/pCount).String())
}

func (ad *ArchiveDownloader) download(file *ArchiveFile) error {
	start := time.Now()
	url := fmt.Sprintf(archiveUrl, file.Year, file.Month, file.Day, file.Hour)

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

				for _, v := range repoIds {
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
		return err //log.Fatalf("failed waiting. error: %v", err)
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
