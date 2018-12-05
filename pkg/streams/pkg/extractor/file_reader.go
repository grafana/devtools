package extractor

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

type githubArchiveFileReader struct {
	path string
}

var NewFileReader = func(path string) (GithubArchiveReaderFactory, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return &githubArchiveFileReader{path: path}, nil
}

func (r *githubArchiveFileReader) NewReaderFor(dt time.Time) (io.ReadCloser, error) {
	fileName := path.Join(r.path, fmt.Sprintf("%s.json.gz", ToGHADate(dt)))
	var reader io.ReadCloser

	if _, err := os.Stat(fileName); os.IsNotExist(err) == true {
		fileName = strings.TrimSuffix(fileName, ".gz")
		reader, err = os.Open(fileName)
		if err != nil {
			fmt.Printf("Failed to read file:\n%v\n", err)
			return nil, err
		}
	} else {
		file, err := os.Open(fileName)
		if err != nil {
			fmt.Printf("Failed to read file:\n%v\n", err)
			return nil, err
		}

		defer file.Close()

		reader, err = gzip.NewReader(file)
		if err != nil {
			fmt.Printf("No data yet, gzip reader:\n%v\n", err)
			return nil, err
		}
	}

	return reader, nil
}

func ToGHADate(dt time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d-%d", dt.Year(), dt.Month(), dt.Day(), dt.Hour())
}
