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

const max_go_routines = 12

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

	startDate := time.Date(2018, time.Month(1), 1, 0, 0, 0, 0, time.Local)
	//stopDate := time.Now()
	stopDate := time.Date(2018, time.Month(1), 2, 12, 0, 0, 0, time.Local)

	urls := buildUrlsDownload(archFiles, startDate, stopDate)
	for _, u := range urls {
		downloadUrls <- u
	}
	close(downloadUrls)

	downloadGroup.Wait()

	log.Println("filtered event: ", len(allEvents))
	log.Println("event count: ", eventCount)
	log.Println("elapsed: ", time.Since(start))
}

func download(file *ArchiveFile) error {
	url := fmt.Sprintf(archiveUrl, file.Year, file.Month, file.Day, file.Hour)
	log.Printf("downloading: %s\n", url)

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

	var events []GithubEventJson

	for {
		zr.Multistream(false)

		scanner := bufio.NewScanner(zr)
		scanner.Buffer([]byte(""), 1024*1024) //increase buffer limit

		for scanner.Scan() {
			ge := GithubEventJson{}
			err := json.Unmarshal([]byte(scanner.Text()), &ge)
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

	engine, err := xorm.NewEngine(database, connectionString)
	engine.SetColumnMapper(core.GonicMapper{})
	if err != nil {
		log.Fatalf("failed to connect to database. error: %v", err)
	}

	var dbEvents []*GithubEvent
	for _, e := range events {
		dbEvents = append(dbEvents, e.CreateGithubEvent())
	}

	// err = insertIntoDatabase(engine, dbEvents)
	// if err != nil {
	// 	log.Fatalf("failed to connect to database. error %v", err)
	// }

	engine.Insert(file)

	lock.Lock()
	allEvents = append(allEvents, events...)
	lock.Unlock()
	return zr.Close()
}

func insertIntoDatabase(engine *xorm.Engine, events []*GithubEvent) error {
	fmt.Println("inserting events", len(events))
	//insert with a batch of 100 rows
	len := len(events)

	i := 0
	//for i := 0; i < len; i += 100 {
	upper := i + 100
	if len-i < 100 {
		upper = len - i
	}

	log.Printf("events: %d %d", i, upper)
	_, err := engine.Insert(events[i:upper])
	if err != nil {
		return errors.Wrap(err, "inserting rows to database")
	}
	//}

	return nil
}
