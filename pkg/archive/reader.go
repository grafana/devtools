package archive

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/grafana/devtools/pkg/common"

	"github.com/go-xorm/xorm"
	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/log"
)

// ArchiveReader reads all events stored in archive database
type ArchiveReader struct {
	logger    log.Logger
	engine    *xorm.Engine
	batchSize int64
}

// NewArchiveReader creates a new reader
func NewArchiveReader(logger log.Logger, engine *xorm.Engine, batchSize int64) *ArchiveReader {
	return &ArchiveReader{
		logger:    logger.New("logger", "archive-reader"),
		engine:    engine,
		batchSize: batchSize,
	}
}

// ReadAllEvents reads all events stored in archive database
func (ar *ArchiveReader) ReadAllEvents() (streams.Readable, <-chan error) {
	r, w := streams.New()
	outErr := make(chan error)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		totalRows := int64(0)
		offset := int64(0)

		totalRows, err := ar.engine.Count(&common.GithubEvent{})
		if err != nil {
			outErr <- err
			return
		}

		totalPages := totalRows / ar.batchSize
		readEvents := int64(0)

		start := time.Now()
		ar.logger.Info("reading events from archive database...", "totalEvents", totalRows, "totalPages", totalPages)

		for n := int64(0); n <= totalPages; n++ {
			if n > 0 {
				offset = n * ar.batchSize
			}

			startBatch := time.Now()
			ar.logger.Debug("reading batch of events from archive database...", "batchSize", ar.batchSize, "offset", offset)
			var rawEvents []*common.GithubEvent
			err := ar.engine.OrderBy("id").Limit(int(ar.batchSize), int(offset)).Find(&rawEvents)
			if err != nil {
				outErr <- err
				return
			}

			ar.logger.Debug("batch of events read from archive database", "batchSize", ar.batchSize, "offset", offset, "eventCount", len(rawEvents), "took", time.Since(startBatch))

			startDecode := time.Now()
			ar.logger.Debug("deserializing json of event batch...")

			for _, rawEvent := range rawEvents {
				reader := strings.NewReader(rawEvent.Data)
				d := json.NewDecoder(reader)
				for {
					var evt ghevents.Event
					err := d.Decode(&evt)
					if err == io.EOF {
						break
					} else if err != nil {
						outErr <- err
						break
					}

					w <- &evt
					readEvents++
				}
			}

			ar.logger.Debug("json of event batch deserialized", "took", time.Since(startDecode))
		}

		ar.logger.Info("events read from archive database", "readEvents", readEvents, "took", time.Since(start))
	}()

	go func() {
		wg.Wait()
		close(w)
		close(outErr)
	}()

	return r, outErr
}
