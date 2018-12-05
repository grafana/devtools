package githubstats

import (
	"time"

	ghevents "github.com/grafana/github-repo-metrics/pkg/streams"
	"github.com/grafana/github-repo-metrics/pkg/streams/pkg/streamprojections"
)

const (
	DailyCommitActivityStream                        = "DailyCommitActvity"
	WeeklyCommitActivityStream                       = "WeeklyCommitActvity"
	MonthlyCommitActivityStream                      = "MonthlyCommitActvity"
	QuarterlyCommitActivityStream                    = "QuarterlyCommitActvity"
	YearlyCommitActivityStream                       = "YearlyCommitActvity"
	CommitActivityStream                             = "CommitActivityStream"
	SevenDaysMovingAverageCommitActivityStream       = "SevenDaysMovingAverageCommitActivity"
	TwentyFourHoursMovingAverageCommitActivityStream = "24HoursMovingAverageCommitActivity"
)

type CommitActivityState struct {
	Time    time.Time `persist:",primarykey"`
	Period  string    `persist:",primarykey"`
	Repo    string    `persist:",primarykey"`
	Commits float64
}

type CommitActivityProjections struct {
	daily                        *streamprojections.StreamProjection
	weekly                       *streamprojections.StreamProjection
	monthly                      *streamprojections.StreamProjection
	quarterly                    *streamprojections.StreamProjection
	yearly                       *streamprojections.StreamProjection
	sevenDaysMovingAverage       *streamprojections.StreamProjection
	twentyFourHoursMovingAverage *streamprojections.StreamProjection
	all                          *streamprojections.StreamProjection
}

func NewCommitActivityProjections() *CommitActivityProjections {
	p := &CommitActivityProjections{}

	p.daily = streamprojections.
		FromStream(PushEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyCommitActivityStream).
		Build()

	p.weekly = streamprojections.
		FromStream(PushEventStream).
		Weekly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyCommitActivityStream).
		Build()

	p.monthly = streamprojections.
		FromStream(PushEventStream).
		Monthly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyCommitActivityStream).
		Build()

	p.quarterly = streamprojections.
		FromStream(PushEventStream).
		Quarterly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyCommitActivityStream).
		Build()

	p.yearly = streamprojections.
		FromStream(PushEventStream).
		Yearly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyCommitActivityStream).
		Build()

	p.sevenDaysMovingAverage = streamprojections.
		FromStream(PushEventStream).
		Daily(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageCommitActivityStream).
		Build()

	p.twentyFourHoursMovingAverage = streamprojections.
		FromStream(PushEventStream).
		Hourly(fromCreatedDate, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		Window(23, 0, "h24", p.applyMovingAverage).
		ToStream(TwentyFourHoursMovingAverageCommitActivityStream).
		Build()

	p.all = streamprojections.
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

func (p *CommitActivityProjections) init(t time.Time, repo, period string) streamprojections.ProjectionState {
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

func (p *CommitActivityProjections) Register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.twentyFourHoursMovingAverage)
	engine.Register(p.all)
}
