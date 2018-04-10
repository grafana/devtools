package main

import (
	"flag"
	"log"
	"strconv"
	"time"

	_ "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

var (
	repo = "grafana/grafana"
	//connectionString = "user=githubstats password=githubstats host=localhost port=5432 dbname=githubstats sslmode=disable", "connection string"
	connectionString = "./test.db"
	apiToken         = ""
	database         = "sqlite3"
	repoIds          = []int64{15111821}
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
	ID     int64
	Type   string
	RepoId int64
	Date   time.Time
}

func (gej *GithubEventJson) CreateGithubEvent() *GithubEvent {
	id, _ := strconv.ParseInt(gej.ID, 10, 0)

	var repoId int64
	if gej.Repo != nil {
		repoId = gej.Repo.ID
	}

	return &GithubEvent{
		ID:     id,
		Type:   gej.Type,
		RepoId: repoId,
		Date:   time.Now(),
	}
}

type GithubEventJson struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Repo *Repo  `json:"repo"`
}

func main() {
	flag.StringVar(&repo, "repo", "grafana/grafana", "name of the repo you want to process")
	//flag.StringVar(&connectionString, "connectionString", "")
	flag.StringVar(&apiToken, "apiToken", "default?", "")
	flag.Parse()

	err := initDatabase(database, connectionString)
	if err != nil {
		log.Fatalf("migration failed. error: %v", err)
	}

	downloadEvents()
}
