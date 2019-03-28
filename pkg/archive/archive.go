package archive

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/grafana/devtools/pkg/common"
	"github.com/grafana/devtools/pkg/streams/log"
	"github.com/pkg/errors"
)

var eventCount int64
var filesWithErrors = []string{}

func (ad *ArchiveDownloader) buildUrlsDownload(archFiles []*common.ArchiveFile, st, stopDate time.Time) []*common.ArchiveFile {
	var result []*common.ArchiveFile

	// create lookup index based in ArchiveFile ID
	index := map[int64]*common.ArchiveFile{}
	for _, a := range archFiles {
		index[a.ID] = a
	}

	for st.Unix() < stopDate.Unix() {
		archivedFile := common.NewArchiveFile(st.Year(), int(st.Month()), st.Day(), st.Hour())
		_, exist := index[archivedFile.ID]
		if !exist {
			result = append(result, archivedFile)
		}

		st = st.Add(time.Hour)
	}

	return result
}

// ArchiveDownloader downloads, decompress and store events for github orgs
type ArchiveDownloader struct {
	logger           log.Logger
	engine           *xorm.Engine
	url              string
	orgNames         []string
	startDate        time.Time
	stopDate         time.Time
	doneChan         chan time.Time
	eventChan        chan *common.GithubEvent
	overrideAllFiles bool
}

// NewArchiveDownloader creates a new downloader
func NewArchiveDownloader(engine *xorm.Engine, overrideAllFiles bool, url string, orgNames []string, startDate, stopDate time.Time, doneChan chan time.Time) *ArchiveDownloader {
	return &ArchiveDownloader{
		logger:           log.New(),
		engine:           engine,
		url:              url,
		orgNames:         orgNames,
		startDate:        startDate,
		stopDate:         stopDate,
		doneChan:         doneChan,
		eventChan:        make(chan *common.GithubEvent, 10),
		overrideAllFiles: overrideAllFiles,
	}
}

func (ad *ArchiveDownloader) SetLogger(logger log.Logger) {
	ad.logger = logger.New("logger", "archive-downloader")
}

func (ad *ArchiveDownloader) spawnWorker(index int, wg *sync.WaitGroup, downloadUrls chan *common.ArchiveFile, done chan bool) {
	wg.Add(1)
	go func(workerID int) {
		defer wg.Done()

		logger := ad.logger.New("workerId", workerID)

		logger.Debug("starting worker")

		for {
			select {
			case <-done:
				logger.Debug("worker is complete")
				return
			case u := <-downloadUrls:
				err := ad.download(u)
				if err != nil {
					filesWithErrors = append(filesWithErrors, fmt.Sprintf("%v", u.CreatedAt))
					logger.Error("failed to download file", "createdAt", u.CreatedAt, "error", err)
				}
			}
		}
	}(index)
}

// DownloadEvents start to download all archive events
func (ad *ArchiveDownloader) DownloadEvents() error {
	start := time.Now()

	ad.logger.Info("downloading events...")

	var archFiles []*common.ArchiveFile
	if !ad.overrideAllFiles {
		ad.logger.Debug("reading stored archive files from the database...")
		err := ad.engine.Find(&archFiles)
		if err != nil {
			return err
		}
		ad.logger.Debug("stored archive files read from the database", "archiveFiles", len(archFiles))
	}

	urls := ad.buildUrlsDownload(archFiles, ad.startDate, ad.stopDate)
	var downloadUrls = make(chan *common.ArchiveFile)
	var done = make(chan bool)
	wg := sync.WaitGroup{}

	// start workers
	for i := 0; i < runtime.NumCPU(); i++ {
		ad.spawnWorker(i, &wg, downloadUrls, done)
	}

	go func() {
		i := 0

		for {
			select {
			case <-ad.doneChan:
				ad.logger.Debug("closing down gracefully, cancelled by parent")
				close(done)
				return
			default:
				if i == len(urls) {
					close(done)
					return
				}
				downloadUrls <- urls[i]
				i++
			}
		}
	}()

	defer close(downloadUrls)

	// wait for all workers to complete
	wg.Wait()

	ad.logger.Info("events downloaded and filtered", "eventCount", eventCount, "fileErrors", len(filesWithErrors), "took", time.Since(start))
	if len(filesWithErrors) > 0 {
		ad.logger.Debug("failed downloads of dates", "dates", strings.Join(filesWithErrors, ","))
	}

	return nil
}

func (ad *ArchiveDownloader) buildDownloadURL(file *common.ArchiveFile) string {
	ft := time.Unix(file.ID, 0).UTC()
	return fmt.Sprintf(ad.url, ft.Year(), ft.Month(), ft.Day(), ft.Hour())
}

func (ad *ArchiveDownloader) download(file *common.ArchiveFile) error {
	start := time.Now()
	ad.logger.Debug("downloading file... ", "date", file.CreatedAt)

	url := ad.buildDownloadURL(file)
	res, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to download json file")
	}

	defer res.Body.Close()

	ad.logger.Debug("file downloaded", "date", file.CreatedAt, "took", time.Since(start))

	// bail out if the request didn't return 200. we will download this file next time
	if res.StatusCode != 200 {
		ad.logger.Warn("bad http status code for file", "date", file.CreatedAt, "statusCode", res.StatusCode)
		return nil
	}

	//create reader that can reader gziped contetn
	zipReader, err := gzip.NewReader(res.Body)
	if err != nil {
		return errors.Wrap(err, "parsing compress content")
	}

	//read all zipped data into memory.
	byteArray, err := ioutil.ReadAll(zipReader)
	if err != nil {
		return err
	}

	defer zipReader.Close()

	inMemReader := bytes.NewReader(byteArray)

	// decompress response body and send to workers.
	bufferReader := bufio.NewReaderSize(inMemReader, 2048*2048)

	var lastError error
	var s string
	for {
		line, isPrefix, err := bufferReader.ReadLine()
		s = ""
		for err == nil {
			if isPrefix {
				s += string(line)
			} else {
				s = string(line)
			}

			if !isPrefix {
				err := ad.parseAndFilterEvent(file, s)
				if err != nil {
					lastError = err
				}
			}

			line, isPrefix, err = bufferReader.ReadLine()
		}

		if err != io.EOF {
			ad.logger.Error("failed to read line from file", "date", file.CreatedAt, "error", err)
			break
		}

		if err == io.EOF {
			break
		}
	}

	//only mark the file as downloaded if all lines are parsed.
	if lastError == nil {
		err = ad.saveFileIntoDatabase(file)
		if err != nil {
			return err
		}
	}

	return lastError
}

func (ad *ArchiveDownloader) parseAndFilterEvent(file *common.ArchiveFile, line string) error {
	ge := common.GithubEventJSON{}
	err := json.Unmarshal([]byte(line), &ge)
	if err != nil {
		ad.logger.Error("failed to parse json of file", "date", file.CreatedAt, "error", err)
		return err
	}

	for _, v := range ad.orgNames {
		if ge.Org != nil && ge.Org.Login == v {
			id, _ := strconv.ParseInt(ge.ID, 10, 0)
			ad.saveEventIntoDatabase(&common.GithubEvent{ID: id, CreatedAt: ge.CreatedAt, Data: string(line)})
			eventCount++
		}
	}

	return nil
}

func (ad *ArchiveDownloader) saveFileIntoDatabase(file *common.ArchiveFile) error {
	//remove the file first to make development easier.
	_, err := ad.engine.Exec("DELETE FROM archive_file WHERE ID = ? ", file.ID)
	if err != nil {
		return err
	}

	_, err = ad.engine.Insert(file)
	return err
}

func (ad *ArchiveDownloader) saveEventIntoDatabase(event *common.GithubEvent) error {
	//remove the event first to make development easier.
	_, err := ad.engine.Exec("DELETE FROM github_event WHERE ID = ? ", event.ID)
	if err != nil {
		return err
	}

	_, err = ad.engine.Insert(event)
	return err
}
