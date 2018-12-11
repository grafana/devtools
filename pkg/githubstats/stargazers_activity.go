package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	DailyStargazersActivityStream     = "d_stars"
	WeeklyStargazersActivityStream    = "w_stars"
	MonthlyStargazersActivityStream   = "m_stars"
	QuarterlyStargazersActivityStream = "q_stars"
	YearlyStargazersActivityStream    = "y_stars"
	StargazersActivityStream          = "all_stars"
)

type StargazersActivityState struct {
	Time   time.Time `persist:",primarykey"`
	Period string    `persist:",primarykey"`
	Repo   string    `persist:",primarykey"`
	Count  float64
}

type StargazersActivityProjections struct {
	daily     *projections.StreamProjection
	weekly    *projections.StreamProjection
	monthly   *projections.StreamProjection
	quarterly *projections.StreamProjection
	yearly    *projections.StreamProjection
	all       *projections.StreamProjection
}

func NewStargazersActivityProjections() *StargazersActivityProjections {
	p := &StargazersActivityProjections{}
	p.daily = projections.
		FromStream(WatchEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "d", p.applyCummalativeSum).
		ToStream(DailyStargazersActivityStream).
		Build()

	p.weekly = projections.
		FromStream(WatchEventStream).
		Weekly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "w", p.applyCummalativeSum).
		ToStream(WeeklyStargazersActivityStream).
		Build()

	p.monthly = projections.
		FromStream(WatchEventStream).
		Monthly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "m", p.applyCummalativeSum).
		ToStream(MonthlyStargazersActivityStream).
		Build()

	p.quarterly = projections.
		FromStream(WatchEventStream).
		Quarterly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "q", p.applyCummalativeSum).
		ToStream(QuarterlyStargazersActivityStream).
		Build()

	p.yearly = projections.
		FromStream(WatchEventStream).
		Yearly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "y", p.applyCummalativeSum).
		ToStream(YearlyStargazersActivityStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyStargazersActivityStream,
			WeeklyStargazersActivityStream,
			MonthlyStargazersActivityStream,
			QuarterlyStargazersActivityStream,
			YearlyStargazersActivityStream,
		).
		ToStream(StargazersActivityStream).
		Persist("stargazers_activity", &StargazersActivityState{}).
		Build()

	return p
}

func (p *StargazersActivityProjections) init(t time.Time, repo, period string) projections.ProjectionState {
	return &StargazersActivityState{
		Time:   t,
		Period: period,
		Repo:   repo,
	}
}

func (p *StargazersActivityProjections) apply(state *StargazersActivityState, evt *ghevents.Event) {
	state.Count++
}

func (p *StargazersActivityProjections) applyCummalativeSum(state *StargazersActivityState, msg *StargazersActivityState, windowSize int) {
	state.Count += msg.Count
}

func (p *StargazersActivityProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.all)
}
