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

//var archiveUrlPattern = `http://localhost:8100/%s-%s-%s-%d.json.gz`
var archiveUrlPattern = `http://data.githubarchive.org/%d-%02d-%02d-%d.json.gz`
var eventCount = 0
var allEvents []GithubEventJson
var lock sync.Mutex

const max_go_routines = 12

func downloadEvents() {
	var downloadUrls = make(chan *ArchiveFile, 0)

	years := []int{2018}
	months := []int{01}
	days := []int{28, 29, 30}

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

	eg := errgroup.Group{}

	for i := 0; i <= max_go_routines; i++ {
		eg.Go(
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

	for _, y := range years {
		for _, m := range months {
			for _, d := range days {
				for hour := 0; hour < 24; hour++ {

					//TODO: use hashset to see if the file have been
					// processed before. This looks like shiet
					af := &ArchiveFile{Year: y, Month: m, Day: d, Hour: hour}
					uni := true
					for _, a := range archFiles {
						if af.Equals(a) {
							uni = false
							break
						}
					}

					if uni {
						downloadUrls <- af
					}

				}
			}
		}
	}

	close(downloadUrls)
	eg.Wait()

	log.Println("filtered event: ", len(allEvents))
	log.Println("event count: ", eventCount)
	log.Println("elapsed: ", time.Since(start))
}

func download(file *ArchiveFile) error {
	url := fmt.Sprintf(archiveUrlPattern, file.Year, file.Month, file.Day, file.Hour)
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

	//insert with a batch of 100 rows
	for i := 0; i > len(dbEvents); i += 100 {
		_, err = engine.Insert(dbEvents[i : i+100])
		if err != nil {
			return errors.Wrap(err, "inserting rows to database")
		}
	}

	engine.Insert(file)

	lock.Lock()
	allEvents = append(allEvents, events...)
	lock.Unlock()
	return zr.Close()
}
