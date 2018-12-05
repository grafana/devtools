package common

import (
	"strconv"
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

type ArchiveFile struct {
	ID        int64
	CreatedAt time.Time
}

func NewArchiveFile(year, month, day, hour int) *ArchiveFile {
	dt := time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.UTC)
	return &ArchiveFile{
		CreatedAt: dt,
		ID:        dt.Unix(),
	}
}

type Repo struct {
	ID int64 `json:"id"`
}

type GithubEvent struct {
	ID        int64
	Type      string
	RepoID    int64
	CreatedAt time.Time
	Payload   *simplejson.Json
	Actor     *simplejson.Json
}

func (gej *GithubEventJSON) CreateGithubEvent() *GithubEvent {
	id, _ := strconv.ParseInt(gej.ID, 10, 0)

	var repoID int64
	if gej.Repo != nil {
		repoID = gej.Repo.ID
	}

	return &GithubEvent{
		ID:        id,
		Type:      gej.Type,
		RepoID:    repoID,
		CreatedAt: gej.CreatedAt,
		Payload:   gej.Payload,
		Actor:     gej.Actor,
	}
}

type GithubEventJSON struct {
	ID        string           `json:"id"`
	Type      string           `json:"type"`
	Repo      *Repo            `json:"repo"`
	Payload   *simplejson.Json `json:"payload"`
	Actor     *simplejson.Json `json:"actor"`
	CreatedAt time.Time        `json:"created_at"`
}

type ActorJSON struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	DisplayLogin string `json:"display_login"`
	GravatarID   string `json:"gravatar_id"`
}
