package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateUrlsFor6Hours(t *testing.T) {
	var archivedFiles []*ArchiveFile

	startDate := time.Date(2018, time.Month(1), 1, 1, 0, 0, 0, time.Local)
	stopDate := time.Date(2018, time.Month(1), 1, 6, 0, 0, 0, time.Local)

	result := buildUrlsDownload(archivedFiles, startDate, stopDate)

	assert.Equal(t, len(result), 6, "failure")
}

func TestGenerateUrlsForOneDay(t *testing.T) {
	var archivedFiles []*ArchiveFile

	startDate := time.Date(2018, time.Month(1), 1, 1, 0, 0, 0, time.Local)
	stopDate := time.Date(2018, time.Month(1), 2, 0, 0, 0, 0, time.Local)

	result := buildUrlsDownload(archivedFiles, startDate, stopDate)

	assert.Equal(t, len(result), 24, "failure")
}

func TestGenerateUrlsForTwoAndAHalfDay(t *testing.T) {
	var archivedFiles []*ArchiveFile

	startDate := time.Date(2018, time.Month(1), 1, 1, 0, 0, 0, time.Local)
	stopDate := time.Date(2018, time.Month(1), 2, 12, 0, 0, 0, time.Local)

	result := buildUrlsDownload(archivedFiles, startDate, stopDate)

	assert.Equal(t, len(result), 36, "failure")
}
