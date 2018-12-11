package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	DailyCommitActivityStream                        = "d_commits"
	WeeklyCommitActivityStream                       = "w_commits"
	MonthlyCommitActivityStream                      = "m_commits"
	QuarterlyCommitActivityStream                    = "q_commits"
	YearlyCommitActivityStream                       = "y_commits"
	SevenDaysMovingAverageCommitActivityStream       = "d7_commits"
	TwentyFourHoursMovingAverageCommitActivityStream = "h24_commits"
	CommitActivityStream                             = "all_commits"
)

type CommitActivityState struct {
	Time    time.Time `persist:",primarykey"`
	Period  string    `persist:",primarykey"`
	Repo    string    `persist:",primarykey"`
	Commits float64
}

type CommitActivityProjections struct {
	daily                        *projections.StreamProjection
	weekly                       *projections.StreamProjection
	monthly                      *projections.StreamProjection
	quarterly                    *projections.StreamProjection
	yearly                       *projections.StreamProjection
	sevenDaysMovingAverage       *projections.StreamProjection
	twentyFourHoursMovingAverage *projections.StreamProjection
	all                          *projections.StreamProjection
}

func NewCommitActivityProjections() *CommitActivityProjections {
	p := &CommitActivityProjections{}

	p.daily = projections.
		FromStream(PushEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyCommitActivityStream).
		Build()

	p.weekly = projections.
		FromStream(PushEventStream).
		Weekly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyCommitActivityStream).
		Build()

	p.monthly = projections.
		FromStream(PushEventStream).
		Monthly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyCommitActivityStream).
		Build()

	p.quarterly = projections.
		FromStream(PushEventStream).
		Quarterly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyCommitActivityStream).
		Build()

	p.yearly = projections.
		FromStream(PushEventStream).
		Yearly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyCommitActivityStream).
		Build()

	p.sevenDaysMovingAverage = projections.
		FromStream(PushEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageCommitActivityStream).
		Build()

	p.twentyFourHoursMovingAverage = projections.
		FromStream(PushEventStream).
		Hourly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(23, 0, "h24", p.applyMovingAverage).
		ToStream(TwentyFourHoursMovingAverageCommitActivityStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyCommitActivityStream,
			WeeklyCommitActivityStream,
			MonthlyCommitActivityStream,
			QuarterlyCommitActivityStream,
			YearlyCommitActivityStream,
			SevenDaysMovingAverageCommitActivityStream,
			TwentyFourHoursMovingAverageCommitActivityStream,
		).
		ToStream(CommitActivityStream).
		Persist("commit_activity", &CommitActivityState{}).
		Build()

	return p
}

func (p *CommitActivityProjections) init(t time.Time, repo, period string) projections.ProjectionState {
	return &CommitActivityState{
		Time:    t,
		Period:  period,
		Repo:    repo,
		Commits: 0,
	}
}

func (p *CommitActivityProjections) apply(state *CommitActivityState, evt *ghevents.Event) {
	if evt.Payload.Ref != nil && *evt.Payload.Ref == "refs/heads/master" && evt.Payload.DistinctSize != nil && *evt.Payload.DistinctSize > 0 {
		state.Commits += float64(*evt.Payload.DistinctSize)
	}
}

func (p *CommitActivityProjections) applyMovingAverage(state *CommitActivityState, msg *CommitActivityState, windowSize int) {
	state.Commits += msg.Commits / float64(windowSize)
}

func (p *CommitActivityProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.twentyFourHoursMovingAverage)
	engine.Register(p.all)
}
