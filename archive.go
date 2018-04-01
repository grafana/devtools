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
)

//

var jsonUrl = `http://localhost:8100/2018-01-01-2.json.gz`

var archiveUrlPattern = `http://localhost:8100/%s-%s-%s-%d.json.gz`

var eventCount = 0
var events []GithubEvent

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
					url := fmt.Sprintf(archiveUrlPattern, y, m, d, hour)
					urls = append(urls, url)
				}
			}
		}

	}

	start := time.Now()
	for _, u := range urls {
		download(u)
	}

	fmt.Println("filtred event: ", len(events))
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

	for {
		zr.Multistream(false)

		//zr2 := bufio.NewReaderSize(zr, 1024*1024)
		scanner := bufio.NewScanner(zr)
		scanner.Buffer([]byte(""), 1024*1024) //increase buffer limit

		for scanner.Scan() {
			ge := GithubEvent{}
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

	logOnError(zr.Close(), "closing buffer")
}
