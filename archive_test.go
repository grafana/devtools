package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var startDate = time.Date(2018, time.Month(1), 1, 0, 0, 0, 0, time.Local)

func TestGenerateUrlsFor6Hours(t *testing.T) {
	var archivedFiles []*ArchiveFile

	testCases := []struct {
		stopDate        time.Time
		expecedUrlCount int
	}{
		{stopDate: fakeStopDate(1, 6), expecedUrlCount: 6},
		{stopDate: fakeStopDate(2, 0), expecedUrlCount: 24},
		{stopDate: fakeStopDate(2, 12), expecedUrlCount: 36},
	}

	for _, tc := range testCases {
		result := buildUrlsDownload(archivedFiles, startDate, tc.stopDate)
		assert.Equal(t, tc.expecedUrlCount, len(result), "failure")
	}
}

func TestGenerateUrlsWhenArchivedFilesExists(t *testing.T) {
	var archivedFiles []*ArchiveFile

	for i := 0; i < 12; i++ {
		archivedFiles = append(archivedFiles, &ArchiveFile{
			Year:  2018,
			Month: 1,
			Day:   1,
			Hour:  i})
	}

	stopDate := fakeStopDate(1, 12)

	t.Run("gap before start date", func(t *testing.T) {
		result := buildUrlsDownload(archivedFiles, startDate, stopDate)
		assert.Equal(t, 12, len(result), "failure")
	})

	t.Run("gap after start date", func(t *testing.T) {
		startDate = startDate.AddDate(0, 0, -1)
		result := buildUrlsDownload(archivedFiles, startDate, stopDate)

		assert.Equal(t, 36, len(result), "failure")
	})
}

func fakeStopDate(days, hours int) time.Time {
	return time.Date(2018, time.Month(1), days, hours, 0, 0, 0, time.Local)
}
