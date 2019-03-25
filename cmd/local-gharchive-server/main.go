package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
)

func main() {
	var (
		srcPath string
		port    string
	)

	flag.StringVar(&srcPath, "path", "", "local path on disk where github archive events are located")
	flag.StringVar(&port, "port", "8000", "port to serve arhive files from")
	flag.Parse()

	withoutGz := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileName := strings.TrimSuffix(r.URL.Path, ".gz")
		if fileName == "/" {
			w.WriteHeader(200)
			return
		}
		fullPath := path.Join(srcPath, fileName)
		log.Println("serving file", "path", fullPath)

		r.URL, _ = url.Parse(fileName)
		w.Header().Add("Content-Type", "application/gzip")

		_, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				w.WriteHeader(404)
				return
			}
			log.Fatal(err)
		}

		var buf bytes.Buffer
		gzipWriter := gzip.NewWriter(&buf)

		f, err := os.Open(fullPath)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(gzipWriter, f)
		if err != nil {
			log.Fatal(err)
		}

		if err := f.Close(); err != nil {
			log.Fatal(err)
		}

		if err := gzipWriter.Close(); err != nil {
			log.Fatal(err)
		}

		reader := bytes.NewReader(buf.Bytes())
		http.ServeContent(w, r, fullPath, time.Now(), reader)
	})

	withGz := gziphandler.GzipHandler(withoutGz)

	http.Handle("/", withGz)
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil)
}
