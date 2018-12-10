package githubstats

import (
	"time"

	ghevents "github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
)

const (
	DailyStargazersActivityStream     = "DailyStargazersActvity"
	WeeklyStargazersActivityStream    = "WeeklyStargazersActvity"
	MonthlyStargazersActivityStream   = "MonthlyStargazersActvity"
	QuarterlyStargazersActivityStream = "QuarterlyStargazersActvity"
	YearlyStargazersActivityStream    = "YearlyStargazersActvity"
	StargazersActivityStream          = "StargazersActivityStream"
)

type StargazersActivityState struct {
	Time   time.Time `persist:",primarykey"`
	Period string    `persist:",primarykey"`
	Repo   string    `persist:",primarykey"`
	Count  float64
}

type StargazersActivityProjections struct {
	daily     *streamprojections.StreamProjection
	weekly    *streamprojections.StreamProjection
	monthly   *streamprojections.StreamProjection
	quarterly *streamprojections.StreamProjection
	yearly    *streamprojections.StreamProjection
	all       *streamprojections.StreamProjection
}

func NewStargazersActivityProjections() *StargazersActivityProjections {
	p := &StargazersActivityProjections{}
	p.daily = streamprojections.
		FromStream(WatchEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "d", p.applyCummalativeSum).
		ToStream(DailyStargazersActivityStream).
		Build()

	p.weekly = streamprojections.
		FromStream(WatchEventStream).
		Weekly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "w", p.applyCummalativeSum).
		ToStream(WeeklyStargazersActivityStream).
		Build()

	p.monthly = streamprojections.
		FromStream(WatchEventStream).
		Monthly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "m", p.applyCummalativeSum).
		ToStream(MonthlyStargazersActivityStream).
		Build()

	p.quarterly = streamprojections.
		FromStream(WatchEventStream).
		Quarterly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "q", p.applyCummalativeSum).
		ToStream(QuarterlyStargazersActivityStream).
		Build()

	p.yearly = streamprojections.
		FromStream(WatchEventStream).
		Yearly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "y", p.applyCummalativeSum).
		ToStream(YearlyStargazersActivityStream).
		Build()

	p.all = streamprojections.
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

func (p *StargazersActivityProjections) init(t time.Time, repo, period string) streamprojections.ProjectionState {
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

func (p *StargazersActivityProjections) Register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.all)
}
