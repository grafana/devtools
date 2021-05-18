package archive

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/grafana/devtools/pkg/common"
	"github.com/grafana/devtools/pkg/streams/log"
	"github.com/pkg/errors"
)

// ArchiveDownloader downloads, decompress and store events for github orgs
type ArchiveDownloader struct {
	logger           log.Logger
	engine           *xorm.Engine
	url              string
	orgNames         []string
	startDate        time.Time
	stopDate         time.Time
	overrideAllFiles bool
	numWorkers       int
	eventCount       int64
	filesWithErrors  []string
	skipErrors       bool
}

// NewArchiveDownloader creates a new downloader
func NewArchiveDownloader(
	engine *xorm.Engine,
	overrideAllFiles bool,
	url string,
	orgNames []string,
	startDate, stopDate time.Time,
	numWorkers int,
	skipErrors bool,
	logger log.Logger,
) *ArchiveDownloader {
	return &ArchiveDownloader{
		logger:           logger.New("logger", "archive-downloader"),
		engine:           engine,
		url:              url,
		orgNames:         orgNames,
		startDate:        startDate,
		stopDate:         stopDate,
		overrideAllFiles: overrideAllFiles,
		numWorkers:       numWorkers,
		filesWithErrors:  make([]string, 0, 16),
		skipErrors:       skipErrors,
	}
}

// DownloadEvents start to download all archive events
func (ad *ArchiveDownloader) DownloadEvents(ctx context.Context) error {
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
	var downloadUrls = make(chan *common.ArchiveFile, 16)
	wg := sync.WaitGroup{}

	// start workers
	for i := 0; i < ad.numWorkers; i++ {
		ad.spawnWorker(ctx, i, &wg, downloadUrls)
	}

	go func() {
		defer close(downloadUrls)

		for _, url := range urls {
			select {
			case downloadUrls <- url:
			case <-ctx.Done():
				ad.logger.Debug("closing down gracefully, cancelled by parent")
				return
			}
		}
	}()

	// wait for all workers to complete
	wg.Wait()

	ad.logger.Info("events downloaded and filtered", "eventCount", ad.eventCount, "fileErrors", len(ad.filesWithErrors), "took", time.Since(start))
	if len(ad.filesWithErrors) > 0 {
		ad.logger.Debug("failed downloads of dates", "dates", strings.Join(ad.filesWithErrors, ","))
	}

	return nil
}

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

func (ad *ArchiveDownloader) spawnWorker(
	ctx context.Context, index int, wg *sync.WaitGroup, downloadUrls chan *common.ArchiveFile,
) {
	wg.Add(1)
	go func(workerID int) {
		defer wg.Done()

		logger := ad.logger.New("workerId", workerID)

		logger.Debug("starting worker")

		for {
			select {
			case <-ctx.Done():
				logger.Debug("worker is complete")
				return
			case u, more := <-downloadUrls:
				if !more {
					return
				}

				if err := ad.download(ctx, u); err != nil {
					ad.filesWithErrors = append(ad.filesWithErrors, fmt.Sprintf("%v", u.CreatedAt))
					logger.Error("failed to download file", "createdAt", u.CreatedAt, "error", err)
				}
			}
		}
	}(index)
}

func (ad *ArchiveDownloader) download(ctx context.Context, file *common.ArchiveFile) error {
	start := time.Now()
	ad.logger.Debug("downloading file...", "date", file.CreatedAt)

	url := ad.buildDownloadURL(file)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
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

	// create reader that can reader gziped content
	zipReader, err := gzip.NewReader(res.Body)
	if err != nil {
		return errors.Wrap(err, "parsing compress content")
	}
	defer zipReader.Close()

	// decompress response body and send to workers.
	bufferReader := bufio.NewReaderSize(zipReader, 2048*2048)

	var (
		s       string
		lastErr error
	)

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
				if err := ad.parseAndFilterEvent(s); err != nil {
					ad.logger.Error("failed to process event", "error", err, "skip", ad.skipErrors)

					if !ad.skipErrors {
						lastErr = err
					}
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

	if lastErr != nil {
		return lastErr
	}

	return ad.saveFileIntoDatabase(file)
}

func (ad *ArchiveDownloader) buildDownloadURL(file *common.ArchiveFile) string {
	ft := time.Unix(file.ID, 0).UTC()
	return fmt.Sprintf(ad.url, ft.Year(), ft.Month(), ft.Day(), ft.Hour())
}

func (ad *ArchiveDownloader) parseAndFilterEvent(line string) error {
	var ge common.GithubEventJSON
	if err := json.Unmarshal([]byte(line), &ge); err != nil {
		return err
	}

	if ge.Org == nil {
		return nil
	}

	for _, v := range ad.orgNames {
		if ge.Org.Login == v {
			id, err := strconv.ParseInt(ge.ID, 10, 0)
			if err != nil {
				return err
			}

			event := &common.GithubEvent{
				ID:        id,
				CreatedAt: ge.CreatedAt,
				Data:      string(line),
			}

			if err := ad.saveEventIntoDatabase(event); err != nil {
				return err
			}

			ad.eventCount++
		}
	}

	return nil
}

func (ad *ArchiveDownloader) saveFileIntoDatabase(file *common.ArchiveFile) error {
	session := ad.engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return err
	}

	//remove the file first to make development easier.
	if _, err := session.Exec("DELETE FROM archive_file WHERE ID = ? ", file.ID); err != nil {
		return err
	}

	if _, err := session.Insert(file); err != nil {
		return err
	}

	return session.Commit()
}

func (ad *ArchiveDownloader) saveEventIntoDatabase(event *common.GithubEvent) error {
	session := ad.engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return err
	}

	//remove the event first to make development easier.
	if _, err := session.Exec("DELETE FROM github_event WHERE ID = ? ", event.ID); err != nil {
		return err
	}

	if _, err := session.Insert(event); err != nil {
		return err
	}

	return session.Commit()
}
