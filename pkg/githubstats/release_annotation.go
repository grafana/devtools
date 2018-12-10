package githubstats

import (
	"strings"
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

type ReleaseAnnotationState struct {
	ID         int       `persist:",primarykey"`
	Time       time.Time `persist:",primarykey"`
	Repo       string
	Title      string
	Tags       string
	Prerelease bool
}

type ReleaseAnnotationProjections struct {
	releaseAnnotation *projections.StreamProjection
}

func NewReleaseAnnotationProjections() *ReleaseAnnotationProjections {
	p := &ReleaseAnnotationProjections{}
	p.releaseAnnotation = projections.
		FromStream(ReleaseEventStream).
		PartitionBy(p.partitionByID).
		Init(p.init).
		Apply(p.apply).
		Persist("release_annotation", &ReleaseAnnotationState{}).
		Build()

	return p
}

func (p *ReleaseAnnotationProjections) partitionByID(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "id", evt.Payload.Release.ID
}

func (p *ReleaseAnnotationProjections) init(id int) projections.ProjectionState {
	return &ReleaseAnnotationState{ID: id}
}

func (p *ReleaseAnnotationProjections) apply(state *ReleaseAnnotationState, evt *ghevents.Event) {
	state.Repo = evt.Repo.Name
	state.Time = *evt.Payload.Release.PublishedAt
	state.Title = *evt.Payload.Release.Name
	state.Tags = evt.Payload.Release.TagName
	state.Prerelease = evt.Payload.Release.Prerelease || strings.Contains(evt.Payload.Release.TagName, "beta") || strings.Contains(evt.Payload.Release.TagName, "rc")
	if state.Prerelease {
		state.Tags += ", prerelase"
	}
}

func (p *ReleaseAnnotationProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.releaseAnnotation)
}
