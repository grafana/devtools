package archive

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
	dt := time.Date(int(year), time.Month(month), int(day), int(hour), 0, 0, 0, time.UTC)
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
	RepoId    int64
	CreatedAt time.Time
	Payload   *simplejson.Json
	Actor     *simplejson.Json
}

func (gej *GithubEventJson) CreateGithubEvent() *GithubEvent {
	id, _ := strconv.ParseInt(gej.ID, 10, 0)

	var repoId int64
	if gej.Repo != nil {
		repoId = gej.Repo.ID
	}

	return &GithubEvent{
		ID:        id,
		Type:      gej.Type,
		RepoId:    repoId,
		CreatedAt: gej.CreatedAt,
		Payload:   gej.Payload,
		Actor:     gej.Actor,
	}
}

type GithubEventJson struct {
	ID        string           `json:"id"`
	Type      string           `json:"type"`
	Repo      *Repo            `json:"repo"`
	Payload   *simplejson.Json `json:"payload"`
	Actor     *simplejson.Json `json:"actor"`
	CreatedAt time.Time        `json:"created_at"`
}

type ActorJson struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	DisplayLogin string `json:"display_login"`
	GravatarId   string `json:"gravatar_id"`
}
