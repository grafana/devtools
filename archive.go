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
	"os"
	"time"

	"github.com/go-xorm/xorm"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var archiveUrlPattern = `http://localhost:8100/%s-%s-%s-%d.json.gz`

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
		download(u)
	}

	log.Println("filtered event: ", len(allEvents))
	log.Println("event count: ", eventCount)
	log.Println("elapsed: ", time.Since(start))
}

func download(url string) {
	fmt.Printf("downloading: %s\n", url)
	res, err := http.Get(url)
	logOnError(err, "failed to download json file")

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)

	zr, err := gzip.NewReader(buf)
	logOnError(err, "parsing compress content")

	var events []GithubEventJson

	for {
		zr.Multistream(false)

		scanner := bufio.NewScanner(zr)
		scanner.Buffer([]byte(""), 1024*1024) //increase buffer limit

		for scanner.Scan() {
			ge := GithubEventJson{}
			err := json.Unmarshal([]byte(scanner.Text()), &ge)
			logOnError(err, "parsing json")

			eventCount++
			if ge.Type == "PushEvent" {
				events = append(events, ge)
			}
		}

		err := scanner.Err()
		if err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			break
		}

		if scanner.Err() == io.EOF {
			fmt.Println("scanner EOF")
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
	for i := 0; i >= len(dbEvents); i += 100 {
		_, err = engine.Insert(dbEvents[i : i+100])
		logOnError(err, "insert")
	}

	allEvents = append(allEvents, events...)
	logOnError(zr.Close(), "closing buffer")
}
