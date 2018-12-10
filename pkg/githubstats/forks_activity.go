package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	DailyForksActivityStream     = "DailyForksActvity"
	WeeklyForksActivityStream    = "WeeklyForksActvity"
	MonthlyForksActivityStream   = "MonthlyForksActvity"
	QuarterlyForksActivityStream = "QuarterlyForksActvity"
	YearlyForksActivityStream    = "YearlyForksActvity"
	ForksActivityStream          = "ForksActivityStream"
)

type ForksActivityState struct {
	Time   time.Time `persist:",primarykey"`
	Period string    `persist:",primarykey"`
	Repo   string    `persist:",primarykey"`
	Count  float64
}

type ForksActivityProjections struct {
	daily     *projections.StreamProjection
	weekly    *projections.StreamProjection
	monthly   *projections.StreamProjection
	quarterly *projections.StreamProjection
	yearly    *projections.StreamProjection
	all       *projections.StreamProjection
}

func NewForksActivityProjections() *ForksActivityProjections {
	p := &ForksActivityProjections{}
	p.daily = projections.
		FromStream(ForkEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "d", p.applyCummalativeSum).
		ToStream(DailyForksActivityStream).
		Build()

	p.weekly = projections.
		FromStream(ForkEventStream).
		Weekly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "w", p.applyCummalativeSum).
		ToStream(WeeklyForksActivityStream).
		Build()

	p.monthly = projections.
		FromStream(ForkEventStream).
		Monthly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "m", p.applyCummalativeSum).
		ToStream(MonthlyForksActivityStream).
		Build()

	p.quarterly = projections.
		FromStream(ForkEventStream).
		Quarterly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "q", p.applyCummalativeSum).
		ToStream(QuarterlyForksActivityStream).
		Build()

	p.yearly = projections.
		FromStream(ForkEventStream).
		Yearly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "y", p.applyCummalativeSum).
		ToStream(YearlyForksActivityStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyForksActivityStream,
			WeeklyForksActivityStream,
			MonthlyForksActivityStream,
			QuarterlyForksActivityStream,
			YearlyForksActivityStream,
		).
		ToStream(ForksActivityStream).
		Persist("forks_activity", &ForksActivityState{}).
		Build()

	return p
}

func (p *ForksActivityProjections) init(t time.Time, repo, period string) projections.ProjectionState {
	return &ForksActivityState{
		Time:   t,
		Period: period,
		Repo:   repo,
	}
}

func (p *ForksActivityProjections) apply(state *ForksActivityState, evt *ghevents.Event) {
	state.Count++
}

func (p *ForksActivityProjections) applyCummalativeSum(state *ForksActivityState, msg *ForksActivityState, windowSize int) {
	state.Count += msg.Count
}

func (p *ForksActivityProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.all)
}
