package ghevents

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"time"
)

var githubArchiveURLFormat = "http://data.githubarchive.org/%s.json.gz"

type ghaReader struct{}

// NewGHAReader returns a new event reader that can read github event files from githubarchive.org.
var NewGHAReader = func() (EventReader, error) {
	return &ghaReader{}, nil
}

func (r *ghaReader) Stats() (*EventReaderStats, error) {
	now := time.Now().UTC()

	if now.Minute() < 10 {
		now = now.Add(-2 * time.Hour)
	} else {
		now = now.Add(-1 * time.Hour)
	}

	now = now.Truncate(60 * time.Minute)

	return &EventReaderStats{
		HasEvents: true,
		Start:     time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		End:       now,
	}, nil
}

func (r *ghaReader) Read(dt time.Time) *EventBatch {
	url := fmt.Sprintf(githubArchiveURLFormat, ToGHADate(dt))
	evtBatch := EventBatch{
		ID:       dt.Unix(),
		Name:     url,
		FromTime: dt,
		Events:   []*ExtractedEvent{},
	}

	resp, err := http.Get(url)
	if err != nil {
		evtBatch.Err = err
		return &evtBatch
	}
	defer resp.Body.Close()

	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		evtBatch.Err = err
		return &evtBatch
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
