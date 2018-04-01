package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	repo             = "grafana/grafana"
	connectionString = ""
	apiToken         = ""
)

type Repo struct {
	Id int64 `json:"id"`
}

type GithubEvent struct {
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
	flag.StringVar(&connectionString, "connectionString", "default?", "connection string")
	flag.StringVar(&apiToken, "apiToken", "default?", "")
	flag.Parse()

	// url := fmt.Sprintf("https://api.github.com/repos/%s/events", repo)
	// keepSearching(url)

	downloadEvents()
}
