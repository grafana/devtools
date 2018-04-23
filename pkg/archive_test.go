package main

import (
	"log"
	"testing"
	"time"

	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/stretchr/testify/assert"
)

var startDate = time.Date(2018, time.Month(1), 1, 0, 0, 0, 0, time.Local)

func TestGenerateUrlsFor6Hours(t *testing.T) {
	var archivedFiles []*ArchiveFile
	ad := &ArchiveDownloader{}

	testCases := []struct {
		stopDate        time.Time
		expecedUrlCount int
	}{
		{stopDate: fakeStopDate(1, 6), expecedUrlCount: 6},
		{stopDate: fakeStopDate(2, 0), expecedUrlCount: 24},
		{stopDate: fakeStopDate(2, 12), expecedUrlCount: 36},
	}

	for _, tc := range testCases {
		result := ad.buildUrlsDownload(archivedFiles, startDate, tc.stopDate)
		assert.Equal(t, tc.expecedUrlCount, len(result), "failure")
	}
}

func TestGenerateUrlsWhenArchivedFilesExists(t *testing.T) {
	var archivedFiles []*ArchiveFile
	ad := &ArchiveDownloader{}

	for i := 0; i < 12; i++ {
		archivedFiles = append(archivedFiles, &ArchiveFile{
			Year:  2018,
			Month: 1,
			Day:   1,
			Hour:  i})
	}

	stopDate := fakeStopDate(1, 12)

	t.Run("gap before start date", func(t *testing.T) {
		result := ad.buildUrlsDownload(archivedFiles, startDate, stopDate)
		assert.Equal(t, 12, len(result), "failure")
	})

	t.Run("gap after start date", func(t *testing.T) {
		startDate = startDate.AddDate(0, 0, -1)
		result := ad.buildUrlsDownload(archivedFiles, startDate, stopDate)

		assert.Equal(t, 36, len(result), "failure")
	})
}

func TestWritingArchiveFile(t *testing.T) {
	file := &ArchiveFile{
		Year:  2018,
		Month: 1,
		Day:   1,
		Hour:  1,
	}

	err := initDatabase("sqlite3", "./test.db")
	if err != nil {
		log.Fatalf("failed to connect to database. error: %v", err)
	}

	engine, err := xorm.NewEngine("sqlite3", "./test.db")
	engine.SetColumnMapper(core.GonicMapper{})

	engine.Insert(file)
}

func TestWritingToDatabase(t *testing.T) {
	ad := &ArchiveDownloader{}

	var eventsToWrite []*GithubEvent

	json, err := simplejson.NewJson([]byte(`{"field": "value"}`))
	if err != nil {
		t.Fatalf("failed to parse json. error: %v", err)
	}
	for i := 1; i < 50; i++ {
		eventsToWrite = append(eventsToWrite, &GithubEvent{
			Payload:   json,
			RepoId:    1,
			CreatedAt: time.Now(),
			Type:      "fakeEvent",
			ID:        int64(i),
		})
	}

	err = initDatabase("sqlite3", "./test.db")
	if err != nil {
		t.Fatalf("failed to connect to database. error: %v", err)
	}

	engine, err := xorm.NewEngine("sqlite3", "./test.db")
	engine.SetColumnMapper(core.GonicMapper{})
	if err != nil {
		t.Fatalf("failed to connect to database. error: %v", err)
	}

	ad.engine = engine
	err = ad.insertIntoDatabase(eventsToWrite)
	if err != nil {
		t.Fatalf("failed to connect to database. error: %v", err)
	}
}

func fakeStopDate(days, hours int) time.Time {
	return time.Date(2018, time.Month(1), days, hours, 0, 0, 0, time.Local)
}
