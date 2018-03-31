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
)

// wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
//

var jsonUrl = `http://localhost:8100/2018-01-01-2.json.gz`

var archiveUrlPattern = `http://localhost:8100/%d-%s-%s-%d.json.gz`

func downloadEvents() {
	var urls []string

	//for day := 0; day < 5; day++ {
	year := 2018
	month := "01"
	day := "01"
	for hour := 1; hour < 24; hour++ {
		url := fmt.Sprintf(archiveUrlPattern, year, month, day, hour)
		urls = append(urls, url)
	}
	//}

	//urls = []string{jsonUrl}

	for _, u := range urls {
		download(u)
	}
}

func download(url string) {
	fmt.Printf("downloading: %s\n", url)
	res, err := http.Get(url)
	logOnError(err, "failed to download json file")

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)

	zr, err := gzip.NewReader(buf)
	logOnError(err, "parsing compress content")

	events := []GithubEvent{}

	for {
		//zr.Multistream(false)
		scanner := bufio.NewScanner(zr)
		for scanner.Scan() {
			ge := GithubEvent{}
			err := json.Unmarshal([]byte(scanner.Text()), &ge)
			logOnError(err, "parsing json")

			if ge.Type == "PushEvent" {
				events = append(events, ge)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			break
		}

		fmt.Println("event count: ", len(events))
		fmt.Print("\n\n")

		if scanner.Err() == io.EOF {
			fmt.Println("scanner EOF")
			break
		}

		err = zr.Reset(buf)
		if err == io.EOF {
			break
		}
		//logOnError(err, "Reseting buffer")
	}

	err = zr.Close()
	logOnError(err, "closing buffer")
}
