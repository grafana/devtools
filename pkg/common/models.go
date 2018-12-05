package common

import (
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
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
	Data      *simplejson.Json
	CreatedAt time.Time
}

// GithubEventJSON is the root json model of an event
type GithubEventJSON struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	//Repo      *RepoJSON        `json:"repo"`
	//Payload   *simplejson.Json `json:"payload"`
	//Actor     *simplejson.Json `json:"actor"`
	Org       *OrgJSON  `json:"org"`
	CreatedAt time.Time `json:"created_at"`
}

// RepoJSON is the json model of an repository
// type RepoJSON struct {
// 	ID   int64  `json:"id"`
// 	Name string `json:"name"`
// }

type OrgJSON struct {
	Login string `json:"login"`
}

// ActorJSON is the json model of an actor
// type ActorJSON struct {
// 	ID           int64  `json:"id"`
// 	Login        string `json:"login"`
// 	DisplayLogin string `json:"display_login"`
// 	GravatarID   string `json:"gravatar_id"`
// }
