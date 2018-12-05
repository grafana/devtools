package ghevents

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type fileReader struct {
	path string
}

type fileWriter struct {
	path              string
	overwriteExisting bool
}

type fileReaderWriter struct {
	reader EventReader
	writer EventWriter
}

// NewFileReader returns a new event reader that can read github event files from file system path.
var NewFileReader = func(path string) (EventReader, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return &fileReader{path: path}, nil
}

// NewFileWriter returns a new event writer that can write github event files to file system path.
var NewFileWriter = func(path string, overwriteExisting bool) (EventWriter, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	} else if err != nil {
		return nil, err
	}

	return &fileWriter{
		path:              path,
		overwriteExisting: overwriteExisting,
	}, nil
}

// NewFileReaderWriter returns a new event reader/writer that can read/write github event files from/to file system path.
var NewFileReaderWriter = func(sourcePath string, destinationPath string, overwriteExisting bool) (EventReaderWriter, error) {
	reader, err := NewFileReader(sourcePath)
	if err != nil {
		return nil, err
	}

	writer, err := NewFileWriter(destinationPath, overwriteExisting)
	if err != nil {
		return nil, err
	}

	return &fileReaderWriter{
		reader: reader,
		writer: writer,
	}, nil
}

var fileRegex = regexp.MustCompile(`(?P<Year>\d{4})-(?P<Month>\d{2})-(?P<Day>\d{2})-(?P<Hour>\d\d*)`)

func (fr *fileReader) Stats() (*EventReaderStats, error) {
	files, err := filepath.Glob(path.Join(fr.path, "*.json"))
	if err != nil {
		return nil, err
	}

	mapSubexpNames := func(m, n []string) map[string]string {
		m, n = m[1:], n[1:]
		r := make(map[string]string, len(m))
		for i := range n {
			r[n[i]] = m[i]
		}
		return r
	}

	epochs := []int64{}

	for _, f := range files {
		m := fileRegex.FindStringSubmatch(f)
		n := fileRegex.SubexpNames()
		r := mapSubexpNames(m, n)

		year, err := strconv.ParseInt(r["Year"], 10, 32)
		if err != nil {
			return nil, err
		}
		month, err := strconv.ParseInt(r["Month"], 10, 32)
		if err != nil {
			return nil, err
		}
		day, err := strconv.ParseInt(r["Day"], 10, 32)
		if err != nil {
			return nil, err
		}
		hour, err := strconv.ParseInt(r["Hour"], 10, 32)
		if err != nil {
			return nil, err
		}

		d2 := time.Date(int(year), time.Month(int(month)), int(day), int(hour), 0, 0, 0, time.UTC)
		found := false
		for _, d := range epochs {
			if d == d2.Unix() {
				found = true
				break
			}
		}

		if !found {
			epochs = append(epochs, d2.Unix())
		}
	}

	sort.Slice(epochs, func(i, j int) bool { return epochs[i] < epochs[j] })

	return &EventReaderStats{
		HasEvents: true,
		Start:     time.Unix(epochs[0], 0).UTC(),
		End:       time.Unix(epochs[len(epochs)-1], 0).UTC(),
	}, nil
}

func (fr *fileReader) Read(dt time.Time) *EventBatch {
	var reader io.ReadCloser
	fileName, compressed := fr.getFileNameForRead(dt)
	evtBatch := EventBatch{
		ID:       dt.Unix(),
		Name:     fileName,
		FromTime: dt,
		Events:   []*ExtractedEvent{},
	}

	reader, err := os.Open(fileName)
	if err != nil {
		evtBatch.Err = err
		return &evtBatch
	}

	if compressed {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			evtBatch.Err = err
			return &evtBatch
		}
	}

	defer reader.Close()

	d := NewDecoder(reader)
	for {
		evt, err := d.Decode()
		if err == io.EOF {
			break
		} else if err != nil {
			evtBatch.Err = err
			return &evtBatch
		}

		fullEventPayload, err := evt.MarshalJSON()
		if err != nil {
			evtBatch.Err = err
			return &evtBatch
		}

		evtBatch.Events = append(evtBatch.Events, &ExtractedEvent{
			ID:               evt.ID,
			Type:             evt.Type,
			Public:           evt.Public,
			CreatedAt:        evt.CreatedAt,
			Actor:            evt.Actor,
			Repo:             evt.Repo,
			Org:              evt.Org,
			FullEventPayload: fullEventPayload,
		})
	}

	return &evtBatch
}

func (fr *fileReader) getFileNameForRead(dt time.Time) (string, bool) {
	compressed := true
	fileName := path.Join(fr.path, fmt.Sprintf("%s.json.gz", ToGHADate(dt)))
	if _, err := os.Stat(fileName); os.IsNotExist(err) == true {
		fileName = strings.TrimSuffix(fileName, ".gz")
		compressed = false
	}

	return fileName, compressed
}

func (fw *fileWriter) Write(dt time.Time, events []*ExtractedEvent) (err error) {
	fileName := fw.getFileNameForWrite(dt)
	var file *os.File

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		file, err = os.Create(fileName)
	} else if !fw.overwriteExisting && os.IsExist(err) {
		file, err = os.Open(fileName)
	} else if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	defer file.Close()

	offset := int64(0)
	for _, e := range events {
		n, err := file.WriteAt(e.FullEventPayload, offset)

		if err != nil {
			return err
		}

		offset = offset + int64(n)
	}

	return nil
}

func (fw *fileWriter) getFileNameForWrite(dt time.Time) string {
	return path.Join(fw.path, fmt.Sprintf("%s.json", ToGHADate(dt)))
}

func (frw *fileReaderWriter) Stats() (*EventReaderStats, error) {
	return frw.reader.Stats()
}

func (frw *fileReaderWriter) Read(dt time.Time) *EventBatch {
	return frw.reader.Read(dt)
}

func (frw *fileReaderWriter) Write(dt time.Time, events []*ExtractedEvent) (err error) {
	return frw.writer.Write(dt, events)
}

func ToGHADate(dt time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d-%d", dt.Year(), dt.Month(), dt.Day(), dt.Hour())
}
