package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-xorm/xorm"
)

var archiveUrlPattern = `http://localhost:8100/%s-%s-%s-%d.json.gz`

var eventCount = 0
var allEvents []GithubEvent

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

	fmt.Println("filtred event: ", len(allEvents))
	fmt.Println("event count: ", eventCount)
	fmt.Println("ellapsed: ", time.Since(start))
}

func download(url string) {
	fmt.Printf("downloading: %s\n", url)
	res, err := http.Get(url)
	logOnError(err, "failed to download json file")

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)

	zr, err := gzip.NewReader(buf)
	logOnError(err, "parsing compress content")

	var events []GithubEvent

	for {
		zr.Multistream(false)

		scanner := bufio.NewScanner(zr)
		scanner.Buffer([]byte(""), 1024*1024) //increase buffer limit

		for scanner.Scan() {
			ge := GithubEvent{}
			err := json.Unmarshal([]byte(scanner.Text()), &ge)
			logOnError(err, "parsing json")

			eventCount++
			if ge.Type == "PushEvent" {
				allEvents = append(allEvents, ge)
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

	x, err := xorm.NewEngine("postgre", "user=githubstats password=githubstats host=localhost port=5432 dbname=githubstats sslmode=disable")

	_, err = x.Insert(events)

	logOnError(err, "insert into db")

	allEvents = append(allEvents, events...)
	logOnError(zr.Close(), "closing buffer")
}
