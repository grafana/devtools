package bots

import (
	"context"
	"fmt"

	"github.com/daniellee/go-github/github"
	githubclient "github.com/grafana/github-repo-metrics/pkg/github-client"
)

// FeatureRequestBot fetches feature requests that are:
// - have state open
// - are older than 60 days
// - do not have a linked PR
type FeatureRequestBot struct {
	ghClient *githubclient.GithubClient
	owner    string
	repo     string
}

// NewFeatureRequestBot creates the bot with your GitHub api key and
// github repo details
func NewFeatureRequestBot(apiToken string, owner string, repo string) *FeatureRequestBot {
	frb := FeatureRequestBot{}
	frb.ghClient = githubclient.NewClient(apiToken)
	frb.owner = owner
	frb.repo = repo

	return &frb
}

// GetStaleFeatureRequests return an array of stale feature requests
func (frb *FeatureRequestBot) GetStaleFeatureRequests(optParams ...string) ([]*github.Issue, error) {
	featureRequestLabel := "type: feature request"
	if len(optParams) > 0 {
		featureRequestLabel = optParams[0]
	}

	issues, err := frb.ghClient.GetStaleFeatureRequests(context.Background(), frb.owner, frb.repo, featureRequestLabel)

	if err != nil {
		return nil, err
	}

	return issues, nil
}

// PrintStaleFeatureRequestsAsCSV prints out the stale feature requests with fmt
// To create a CSV From the cmd line:
// go run cmd/main.go -apiToken=xxx  > /tmp/stalefrs.csv
func (frb *FeatureRequestBot) PrintStaleFeatureRequestsAsCSV() error {
	issues, err := frb.GetStaleFeatureRequests()
	if err != nil {
		return err
	}

	fmt.Println("Issue Nr;Title;CreatedAt;URL;")
	for _, issue := range issues {
		fmt.Printf("%v;%v;%v;https://github.com/grafana/grafana/issues/%v;", *issue.Number, *issue.Title, *issue.CreatedAt, *issue.Number)
		fmt.Println()
	}

	return nil
}
