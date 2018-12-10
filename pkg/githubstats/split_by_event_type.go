package githubstats

import (
	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	GithubEventStream       = "github_events"
	IssuesEventStream       = "IssuesEvent"
	PullRequestEventStream  = "PullRequestEvent"
	IssueCommentEventStream = "IssueCommentEvent"
	PushEventStream         = "PushEvent"
	ReleaseEventStream      = "ReleaseEvent"
	ForkEventStream         = "ForkEvent"
	WatchEventStream        = "WatchEvent"
)

type SplitByEventTypeProjections struct {
	split *projections.StreamProjection
}

func NewSplitByEventTypeProjections() *SplitByEventTypeProjections {
	p := &SplitByEventTypeProjections{}
	p.split = projections.
		FromStream(GithubEventStream).
		Filter(p.patchIncorrectRepos).
		ToStreams(p.toStreams).
		Build()

	return p
}

func (p *SplitByEventTypeProjections) patchIncorrectRepos(msg interface{}) bool {
	evt := msg.(*ghevents.Event)
	if evt.Repo.Name == "grafana/" {
		evt.Repo.Name = "grafana/grafana"
	}

	return true
}

func (p *SplitByEventTypeProjections) toStreams(state []projections.ProjectionState) map[string][]projections.ProjectionState {
	dict := map[string][]projections.ProjectionState{}

	for _, item := range state {
		evt := item.(*ghevents.Event)
		if _, ok := dict[evt.Type]; !ok {
			dict[evt.Type] = []projections.ProjectionState{}
		}
		dict[evt.Type] = append(dict[evt.Type], item)
	}

	return dict
}

func (p *SplitByEventTypeProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.split)
}
