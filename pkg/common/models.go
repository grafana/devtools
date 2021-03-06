package common

import (
	"time"
)

// ArchiveFile is the database model for storing reference to an archive file
type ArchiveFile struct {
	ID        int64
	CreatedAt time.Time
}

// NewArchiveFile creates a new archive file
func NewArchiveFile(year, month, day, hour int) *ArchiveFile {
	dt := time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.UTC)
	return &ArchiveFile{
		CreatedAt: dt,
		ID:        dt.Unix(),
	}
}

// GithubEvent is the database model of an event
type GithubEvent struct {
	ID        int64
	Data      string
	CreatedAt time.Time
}

// GithubEventJSON is the root json model of an event
type GithubEventJSON struct {
	ID        string    `json:"id"`
	Org       *OrgJSON  `json:"org"`
	CreatedAt time.Time `json:"created_at"`
}

// OrgJSON is the json model from archive events
type OrgJSON struct {
	Login string `json:"login"`
}
