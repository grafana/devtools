package githubclient

import (
	"context"

	"time"

	"github.com/daniellee/go-github/github"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	Client *github.Client
}

// NewClient is the constructor method for the GitHub client wrapper
func NewClient(accessToken string) *GithubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)

	tc := oauth2.NewClient(oauth2.NoContext, ts)
	ghc := GithubClient{}
	ghc.Client = github.NewClient(tc)

	return &ghc
}

// GetAllPullRequests - used for the monthly report
func (ghc *GithubClient) GetAllPullRequestsForMonth(ctx context.Context, owner string, repo string, month time.Month) ([]*github.Issue, error) {
	opt := &github.IssueListByRepoOptions{State: "all", Since: time.Date(2018, month, 1, 0, 0, 0, 0, time.UTC)}
	opt.ListOptions = github.ListOptions{PerPage: 100}
	var allIssues []*github.Issue

	for {
		issues, resp, err := ghc.Client.Issues.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}
		for _, issue := range issues {
			if issue.PullRequestLinks != nil {
				allIssues = append(allIssues, issue)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}

	return allIssues, nil
}

// GetStaleFeatureRequests returns a list of issues with the feature request label that are:
// - older than 60 days
// - have less than 5 positive reactions
// - have no cross referenced PR's
func (ghc *GithubClient) GetStaleFeatureRequests(ctx context.Context, owner string, repo string, frLabel string) ([]*github.Issue, error) {
	minimumReactions := 5
	var staleAfterDays time.Duration = 60

	opt := &github.IssueListByRepoOptions{State: "all", Since: time.Date(2013, time.January, 1, 0, 0, 0, 0, time.UTC)}
	opt.ListOptions = github.ListOptions{PerPage: 100}
	opt.Labels = []string{frLabel}
	opt.State = "open"
	var allIssues []*github.Issue

	for {
		issues, resp, err := ghc.Client.Issues.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}
		for _, issue := range issues {
			if issue.Reactions != nil && (*issue.Reactions.Heart+*issue.Reactions.PlusOne < minimumReactions) && issue.CreatedAt.Before(time.Now().Add(-staleAfterDays*24*time.Hour)) {
				if !ghc.hasCrossReferencedPR(ctx, owner, repo, *issue.Number) {
					allIssues = append(allIssues, issue)
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}

	return allIssues, nil
}

func (ghc *GithubClient) hasCrossReferencedPR(ctx context.Context, owner string, repo string, issueNumber int) bool {
	timeLine, _, err := ghc.Client.Issues.ListIssueTimeline(ctx, owner, repo, issueNumber, &github.ListOptions{PerPage: 100})
	if err != nil {
		return false
	}

	for _, timeLineEvent := range timeLine {
		if *timeLineEvent.Event == "cross-referenced" && timeLineEvent.Source.Issue != nil && timeLineEvent.Source.Issue.PullRequestLinks != nil {
			return true
		}
	}

	return false
}
