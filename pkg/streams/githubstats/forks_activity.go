package githubstats

import (
	"time"

	ghevents "github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
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
	daily     *streamprojections.StreamProjection
	weekly    *streamprojections.StreamProjection
	monthly   *streamprojections.StreamProjection
	quarterly *streamprojections.StreamProjection
	yearly    *streamprojections.StreamProjection
	all       *streamprojections.StreamProjection
}

func NewForksActivityProjections() *ForksActivityProjections {
	p := &ForksActivityProjections{}
	p.daily = streamprojections.
		FromStream(ForkEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "d", p.applyCummalativeSum).
		ToStream(DailyForksActivityStream).
		Build()

	p.weekly = streamprojections.
		FromStream(ForkEventStream).
		Weekly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "w", p.applyCummalativeSum).
		ToStream(WeeklyForksActivityStream).
		Build()

	p.monthly = streamprojections.
		FromStream(ForkEventStream).
		Monthly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "m", p.applyCummalativeSum).
		ToStream(MonthlyForksActivityStream).
		Build()

	p.quarterly = streamprojections.
		FromStream(ForkEventStream).
		Quarterly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "q", p.applyCummalativeSum).
		ToStream(QuarterlyForksActivityStream).
		Build()

	p.yearly = streamprojections.
		FromStream(ForkEventStream).
		Yearly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(-1, 0, "y", p.applyCummalativeSum).
		ToStream(YearlyForksActivityStream).
		Build()

	p.all = streamprojections.
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

func (p *ForksActivityProjections) init(t time.Time, repo, period string) streamprojections.ProjectionState {
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

func (p *ForksActivityProjections) Register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.all)
}
