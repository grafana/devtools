package archive

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/grafana/devtools/pkg/common"

	"github.com/go-xorm/xorm"
	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams"
)

// ArchiveReader reads all events stored in archive database
type ArchiveReader struct {
	engine    *xorm.Engine
	batchSize int64
}

// NewArchiveReader creates a new reader
func NewArchiveReader(engine *xorm.Engine, batchSize int64) *ArchiveReader {
	return &ArchiveReader{
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

		log.Println("Reading events...", "total rows", totalRows, "total pages", totalPages)

		for n := int64(0); n <= totalPages; n++ {
			if n > 0 {
				offset = n * ar.batchSize
			}

			var rawEvents []*common.GithubEvent
			err := ar.engine.OrderBy("id").Limit(int(ar.batchSize), int(offset)).Find(&rawEvents)
			if err != nil {
				outErr <- err
				return
			}

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
		}

		log.Println("Reading events DONE", "read events", readEvents)
	}()

	go func() {
		wg.Wait()
		close(w)
		close(outErr)
	}()

	return r, outErr
}
