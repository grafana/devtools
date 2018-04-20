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

	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var eventCount = 0
var allEvents []GithubEventJson
var lock sync.Mutex

const max_go_routines = 2

var dlTotal int64
var dlCount int64
var pTotal int64
var pCount int64

func buildUrlsDownload(archFiles []*ArchiveFile, startDate, stopDate time.Time) []*ArchiveFile {
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

func downloadEvents() {
	var downloadUrls = make(chan *ArchiveFile, 0)
	start := time.Now()

	engine, err := xorm.NewEngine(database, connectionString)
	engine.SetColumnMapper(core.GonicMapper{})
	if err != nil {
		log.Fatalf("cannot setup connection. error: %v", err)
	}

	var archFiles []*ArchiveFile
	err = engine.Find(&archFiles)
	if err != nil {
		log.Fatalf("could not find archivefile. error: %v", err)
	}

	log.Printf("found %v arch files", len(archFiles))

	downloadGroup := errgroup.Group{}

	for i := 0; i <= max_go_routines; i++ {
		downloadGroup.Go(
			func() error {
				for u := range downloadUrls {
					err := download(u)
					if err != nil {
						log.Fatalf("failed to download file. error: %v", err)
					}
				}
				return nil
			})
	}

	startDate := time.Date(2015, time.Month(1), 1, 0, 0, 0, 0, time.Local)
	//stopDate := time.Now()
	//stopDate := time.Date(2018, time.Month(1), 3, 12, 0, 0, 0, time.Local)
	stopDate := time.Date(2015, time.Month(1), 2, 0, 0, 0, 0, time.Local)

	urls := buildUrlsDownload(archFiles, startDate, stopDate)
	for _, u := range urls {
		downloadUrls <- u
	}
	close(downloadUrls)

	downloadGroup.Wait()

	log.Println("filtered event: ", len(allEvents))
	log.Println("event count: ", eventCount)
	log.Println("elapsed: ", time.Since(start))

	log.Println("avg dl ", time.Duration(dlTotal/dlCount).String())
	log.Println("avg p ", time.Duration(pTotal/pCount).String())
}

func download(file *ArchiveFile) error {
	start := time.Now()

	url := fmt.Sprintf(archiveUrl, file.Year, file.Month, file.Day, file.Hour)
	log.Printf("downloading: %s\n", url)

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
	for i := 0; i <= 6; i++ {
		eg.Go(func() error {
			for line := range lines {
				ge := GithubEventJson{}
				err := json.Unmarshal(line, &ge)
				if err != nil {
					return errors.Wrap(err, "parsing json")
				}

				eventCount++
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

	err = eg.Wait()
	if err != nil {
		log.Fatalf("failed waiting. error: %v", err)
	}

	engine, err := xorm.NewEngine(database, connectionString)
	engine.SetColumnMapper(core.GonicMapper{})
	if err != nil {
		log.Fatalf("failed to connect to database. error: %v", err)
	}

	var dbEvents []*GithubEvent
	for _, e := range events {
		dbEvents = append(dbEvents, e.CreateGithubEvent())
	}

	err = insertIntoDatabase(engine, dbEvents)
	if err != nil {
		log.Fatalf("failed to connect to database. error %v", err)
	}

	engine.Insert(file)

	pTotal += int64(time.Now().Sub(pStart))
	pCount += 1

	lock.Lock()
	allEvents = append(allEvents, events...)
	lock.Unlock()
	return zr.Close()
}

func insertIntoDatabase(engine *xorm.Engine, events []*GithubEvent) error {

	// we could batch this if we need to write things faster.
	for _, e := range events {
		_, err := engine.Exec("DELETE FROM github_event WHERE ID = ? ", e.ID)
		if err != nil {
			return err
		}

		_, err = engine.Insert(e)
		if err != nil {
			return err
		}
	}

	return nil
}
