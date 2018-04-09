package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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
	repoId           = int64(15111821)
)

type Repo struct {
	Id int64 `json:"id"`
}

type GithubEvent struct {
	Id     int64
	Type   string
	RepoId int64
	Date   time.Time
}

func (gej *GithubEventJson) CreateGithubEvent() *GithubEvent {
	id, _ := strconv.ParseInt(gej.Id, 10, 0)

	var repoId int64
	if gej.Repo != nil {
		repoId = gej.Repo.Id
	}

	return &GithubEvent{
		Id:     id,
		Type:   gej.Type,
		RepoId: repoId,
		Date:   time.Now(),
	}
}

type GithubEventJson struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Repo *Repo  `json:"repo"`
}

func logOnError(err error, message string, args ...interface{}) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, message+" error: %v", append(args, err))
	os.Exit(1)
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
