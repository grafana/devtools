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
	"time"

	"github.com/go-xorm/xorm"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

//var archiveUrlPattern = `http://localhost:8100/%s-%s-%s-%d.json.gz`
var archiveUrlPattern = `http://data.githubarchive.org/%s-%s-%s-%d.json.gz`
var eventCount = 0
var allEvents []GithubEventJson

func downloadEvents() {
	var urls []string

	years := []string{"2018"}
	months := []string{"01"}
	//days := []string{"01", "02", "03", "04", "05"}
	days := []string{"01"}
	for _, y := range years {
		for _, m := range months {
			for _, d := range days {
				for hour := 1; hour < 24; hour++ {
					urls = append(urls, fmt.Sprintf(archiveUrlPattern, y, m, d, hour))
				}
			}
		}
	}

	start := time.Now()
	for _, u := range urls {
		err := download(u)
		if err != nil {
			log.Fatalf("failed to download file. error: %v", err)
		}
	}

	log.Println("filtered event: ", len(allEvents))
	log.Println("event count: ", eventCount)
	log.Println("elapsed: ", time.Since(start))
}

func download(url string) error {
	log.Printf("downloading: %s\n", url)
	res, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to download json file")
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)

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
			//if ge.Type == "PushEvent" {
			if ge.Repo.Id == repoId {
				events = append(events, ge)
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

	allEvents = append(allEvents, events...)
	return zr.Close()
}
