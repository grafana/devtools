package archive

import (
	"testing"
	"time"

	"github.com/grafana/devtools/pkg/common"
	"github.com/stretchr/testify/assert"
)

var startDate = time.Date(2018, time.Month(1), 1, 0, 0, 0, 0, time.UTC)

func TestGenerateUrlsFor6Hours(t *testing.T) {
	var archivedFiles []*common.ArchiveFile
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
	var archivedFiles []*common.ArchiveFile
	ad := &ArchiveDownloader{}

	for i := 0; i < 12; i++ {
		archivedFiles = append(archivedFiles, common.NewArchiveFile(2018, 1, 1, i))
	}

	extraHours := 23
	stopDate := time.Date(2018, time.Month(1), 1, extraHours, 0, 0, 0, time.UTC)

	t.Run("gap before start date", func(t *testing.T) {
		result := ad.buildUrlsDownload(archivedFiles, startDate, stopDate)
		assert.Equal(t, extraHours-12, len(result), "failure")
	})

	t.Run("gap after start date", func(t *testing.T) {
		startDate = startDate.AddDate(0, 0, -2) //start two days earlier
		result := ad.buildUrlsDownload(archivedFiles, startDate, stopDate)

		assert.Equal(t, 48+extraHours-12, len(result), "should get events for two days + 12 hours")
	})
}

func TestWritingArchiveFile(t *testing.T) {
	engine, err := InitDatabase("sqlite3", "./test.db")
	if err != nil {
		t.Fatalf("error initialising database: %v", err)
	}

	ad := &ArchiveDownloader{
		engine: engine,
	}

	assert.NoError(t, ad.saveFileIntoDatabase(common.NewArchiveFile(2018, 1, 1, 1)))
}

func TestWritingToDatabase(t *testing.T) {
	engine, err := InitDatabase("sqlite3", "./test.db")
	if err != nil {
		t.Fatalf("error initialising database: %v", err)
	}

	ad := &ArchiveDownloader{
		engine: engine,
	}

	eventsToWrite := make([]*common.GithubEvent, 0, 50)
	for i := 1; i < 50; i++ {
		eventsToWrite = append(eventsToWrite, &common.GithubEvent{
			Data:      `{"field": "value"}`,
			CreatedAt: time.Now(),
			ID:        int64(i),
		})
	}

	for _, etw := range eventsToWrite {
		assert.NoError(t, ad.saveEventIntoDatabase(etw))
	}
}

func TestArchiveFileIdFormat(t *testing.T) {
	af := common.NewArchiveFile(2015, 10, 10, 10)

	dt := time.Date(2015, time.Month(10), 10, 10, 0, 0, 0, time.UTC)

	assert.Equal(t, af.ID, dt.Unix(), af.ID)
}

func fakeStopDate(days, hours int) time.Time {
	return time.Date(2018, time.Month(1), days, hours, 0, 0, 0, time.UTC)
}
