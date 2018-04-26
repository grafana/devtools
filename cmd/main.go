package main

import (
	"flag"
	"log"

	"github.com/grafana/github-repo-metrics/cmd/bots"
)

var (
	owner    = "grafana"
	repo     = "grafana"
	apiToken = ""
)

func main() {
	flag.StringVar(&owner, "owner", "grafana", "Name of the org you want to process")
	flag.StringVar(&repo, "repo", "grafana", "Name of the repo you want to process")
	flag.StringVar(&apiToken, "apiToken", "default?", "GitHub apiToken")
	flag.Parse()

	frb := bots.NewFeatureRequestBot(apiToken, owner, repo)
	err := frb.PrintStaleFeatureRequestsAsCSV()

	if err != nil {
		log.Fatalf("Failed to fetch stale feature requests. Error: %v", err)
	}
}
