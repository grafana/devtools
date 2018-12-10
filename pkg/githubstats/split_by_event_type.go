package githubstats

import (
	ghevents "github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
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
	split *streamprojections.StreamProjection
}

func NewSplitByEventTypeProjections() *SplitByEventTypeProjections {
	p := &SplitByEventTypeProjections{}
	p.split = streamprojections.
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

func (p *SplitByEventTypeProjections) toStreams(state []streamprojections.ProjectionState) map[string][]streamprojections.ProjectionState {
	dict := map[string][]streamprojections.ProjectionState{}

	for _, item := range state {
		evt := item.(*ghevents.Event)
		if _, ok := dict[evt.Type]; !ok {
			dict[evt.Type] = []streamprojections.ProjectionState{}
		}
		dict[evt.Type] = append(dict[evt.Type], item)
	}

	return dict
}

func (p *SplitByEventTypeProjections) Register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.split)
}
