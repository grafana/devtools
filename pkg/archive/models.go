package archive

import (
	"strconv"
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

type ArchiveFile struct {
	ID    int64
	Year  int
	Month int
	Day   int
	Hour  int
}

// Equals compare two *ArchiveFile to see if they are equal
func (a *ArchiveFile) Equals(other *ArchiveFile) bool {
	return a.Year == other.Year &&
		a.Month == other.Month &&
		a.Day == other.Day &&
		a.Hour == other.Hour
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
	//Url string `json:"url"`
	//AvatarUrl string `json:"avatar_url"`
}
