package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	DailyIssuesOpenClosedActivityStream     = "DailyIssuesOpenClosedActvity"
	WeeklyIssuesOpenClosedActivityStream    = "WeeklyIssuesOpenClosedActvity"
	MonthlyIssuesOpenClosedActivityStream   = "MonthlyIssuesOpenClosedActvity"
	QuarterlyIssuesOpenClosedActivityStream = "QuarterlyIssuesOpenClosedActvity"
	YearlyIssuesOpenClosedActivityStream    = "YearlyIssuesOpenClosedActvity"
	IssuesOpenClosedActivityStream          = "IssuesOpenClosedActivityStream"
)

type IssuesOpenClosedActivityState struct {
	Time           time.Time `persist:",primarykey"`
	Period         string    `persist:",primarykey"`
	Repo           string    `persist:",primarykey"`
	Open           float64
	Closed         float64
	BugsOpen       float64 `persist:"bugs_open"`
	BugsClosed     float64 `persist:"bugs_closed"`
	FeaturesOpen   float64 `persist:"features_open"`
	FeaturesClosed float64 `persist:"features_closed"`
	isOpen         bool
}

type IssuesOpenClosedActivityProjections struct {
	daily     *projections.StreamProjection
	weekly    *projections.StreamProjection
	monthly   *projections.StreamProjection
	quarterly *projections.StreamProjection
	yearly    *projections.StreamProjection
	all       *projections.StreamProjection
}

func NewIssuesOpenClosedActivityProjections() *IssuesOpenClosedActivityProjections {
	p := &IssuesOpenClosedActivityProjections{}
	p.daily = projections.
		FromStream(IssuesEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "d", p.applyCummalativeSum).
		ToStream(DailyIssuesOpenClosedActivityStream).
		Build()

	p.weekly = projections.
		FromStream(IssuesEventStream).
		Weekly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "w", p.applyCummalativeSum).
		ToStream(WeeklyIssuesOpenClosedActivityStream).
		Build()

	p.monthly = projections.
		FromStream(IssuesEventStream).
		Monthly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "m", p.applyCummalativeSum).
		ToStream(MonthlyIssuesOpenClosedActivityStream).
		Build()

	p.quarterly = projections.
		FromStream(IssuesEventStream).
		Quarterly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "q", p.applyCummalativeSum).
		ToStream(QuarterlyIssuesOpenClosedActivityStream).
		Build()

	p.yearly = projections.
		FromStream(IssuesEventStream).
		Yearly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "y", p.applyCummalativeSum).
		ToStream(YearlyIssuesOpenClosedActivityStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyIssuesOpenClosedActivityStream,
			WeeklyIssuesOpenClosedActivityStream,
			MonthlyIssuesOpenClosedActivityStream,
			QuarterlyIssuesOpenClosedActivityStream,
			YearlyIssuesOpenClosedActivityStream,
		).
		ToStream(IssuesOpenClosedActivityStream).
		Persist("issues_open_closed", &IssuesOpenClosedActivityState{}).
		Build()

	return p
}

func (p *IssuesOpenClosedActivityProjections) init(t time.Time, repo, period string) projections.ProjectionState {
	return &IssuesOpenClosedActivityState{
		Time:   t,
		Period: period,
		Repo:   repo,
	}
}

func (p *IssuesOpenClosedActivityProjections) apply(state *IssuesOpenClosedActivityState, evt *ghevents.Event) {
	isFeature := false
	isBug := false

	for _, l := range evt.Payload.Issue.Labels {
		if l.Name == "type: feature request" || l.Name == "type: feature" || l.Name == "type: new feature request" {
			isFeature = true
			continue
		}

		if l.Name == "type: bug" {
			isBug = true
			continue
		}
	}

	switch *evt.Payload.Action {
	case "opened":
		state.isOpen = true
		state.Open++

		if isBug {
			state.BugsOpen++
		}
		if isFeature {
			state.FeaturesOpen++
		}
	case "closed":
		state.isOpen = false
		state.Closed++

		if isBug {
			state.BugsClosed++
		}
		if isFeature {
			state.FeaturesClosed++
		}
	case "reopened":
		state.isOpen = true
		state.Open++
		state.Closed--

		if isBug {
			state.BugsOpen++
			state.BugsClosed--
		}
		if isFeature {
			state.FeaturesOpen++
			state.FeaturesClosed--
		}
	case "labeled":
		{
			if state.isOpen {
				if isBug {
					state.BugsOpen++
				}
				if isFeature {
					state.FeaturesOpen++
				}
			}

			if !state.isOpen {
				if isBug {
					state.BugsClosed++
				}
				if isFeature {
					state.FeaturesClosed++
				}
			}
		}
	case "unlabeled":
		{
			if state.isOpen {
				if isBug {
					state.BugsOpen--
				}
				if isFeature {
					state.FeaturesOpen--
				}
			}

			if !state.isOpen {
				if isBug {
					state.BugsClosed--
				}
				if isFeature {
					state.FeaturesClosed--
				}
			}
		}
	}
}

func (p *IssuesOpenClosedActivityProjections) applyCummalativeSum(state *IssuesOpenClosedActivityState, msg *IssuesOpenClosedActivityState, windowSize int) {
	state.Open += (msg.Open - msg.Closed)
	state.Closed += msg.Closed
	state.BugsOpen += (msg.BugsOpen - msg.BugsClosed)
	state.BugsClosed += msg.BugsClosed
	state.FeaturesOpen += (msg.FeaturesOpen - msg.FeaturesClosed)
	state.FeaturesClosed += msg.FeaturesClosed
}

func (p *IssuesOpenClosedActivityProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.all)
}
