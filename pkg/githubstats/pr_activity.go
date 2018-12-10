package githubstats

import (
	"time"

	ghevents "github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
)

const (
	DailyPullRequestActivityStream                  = "DailyPullRequestActivity"
	WeeklyPullRequestActivityStream                 = "WeeklyPullRequestActivity"
	MonthlyPullRequestActivityStream                = "MonthlyPullRequestActivity"
	QuarterlyPullRequestActivityStream              = "QuarterlyPullRequestActivity"
	YearlyPullRequestActivityStream                 = "YearlyPullRequestActivity"
	PullRequestActivityStream                       = "PullRequestActivityStream"
	SevenDaysMovingAveragePullRequestActivityStream = "SevenDaysMovingAveragePullRequestActivity"
)

type PullRequestActivityState struct {
	Time                      time.Time `persist:",primarykey"`
	Period                    string    `persist:",primarykey"`
	Repo                      string    `persist:",primarykey"`
	ProposedBy                string    `persist:"proposed_by,primarykey"`
	Opened                    float64
	Merged                    float64
	ClosedWithUnmergedCommits float64 `persist:"closed_with_unmerged_commits"`
}

type PullRequestActivityProjections struct {
	daily                  *streamprojections.StreamProjection
	sevenDaysMovingAverage *streamprojections.StreamProjection
	weekly                 *streamprojections.StreamProjection
	monthly                *streamprojections.StreamProjection
	quarterly              *streamprojections.StreamProjection
	yearly                 *streamprojections.StreamProjection
	all                    *streamprojections.StreamProjection
}

func NewPullRequestActivityProjections() *PullRequestActivityProjections {
	p := &PullRequestActivityProjections{}
	p.daily = streamprojections.
		FromStream(PullRequestEventStream).
		Filter(filterByOpenedAndClosedActions).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByPrAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyPullRequestActivityStream).
		Build()

	p.weekly = streamprojections.
		FromStream(PullRequestEventStream).
		Filter(filterByOpenedAndClosedActions).
		Weekly(fromCreatedDate, partitionByRepo, p.partitionByPrAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyPullRequestActivityStream).
		Build()

	p.monthly = streamprojections.
		FromStream(PullRequestEventStream).
		Filter(filterByOpenedAndClosedActions).
		Monthly(fromCreatedDate, partitionByRepo, p.partitionByPrAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyPullRequestActivityStream).
		Build()

	p.quarterly = streamprojections.
		FromStream(PullRequestEventStream).
		Filter(filterByOpenedAndClosedActions).
		Quarterly(fromCreatedDate, partitionByRepo, p.partitionByPrAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyPullRequestActivityStream).
		Build()

	p.yearly = streamprojections.
		FromStream(PullRequestEventStream).
		Filter(filterByOpenedAndClosedActions).
		Yearly(fromCreatedDate, partitionByRepo, p.partitionByPrAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyPullRequestActivityStream).
		Build()

	p.sevenDaysMovingAverage = streamprojections.
		FromStream(PullRequestEventStream).
		Filter(filterByOpenedAndClosedActions).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByPrAuthor).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAveragePullRequestActivityStream).
		Build()

	p.all = streamprojections.
		FromStreams(
			DailyPullRequestActivityStream,
			WeeklyPullRequestActivityStream,
			MonthlyPullRequestActivityStream,
			QuarterlyPullRequestActivityStream,
			YearlyPullRequestActivityStream,
			SevenDaysMovingAveragePullRequestActivityStream,
		).
		ToStream(PullRequestActivityStream).
		Persist("pr_activity", &PullRequestActivityState{}).
		Build()

	return p
}

func (p *PullRequestActivityProjections) partitionByPrAuthor(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "proposedBy", mapUserLoginToGroup(evt.Payload.PullRequest.User.Login)
}

func (p *PullRequestActivityProjections) init(t time.Time, repo, contributorGroup, period string) streamprojections.ProjectionState {
	return &PullRequestActivityState{
		Time:       t,
		Period:     period,
		Repo:       repo,
		ProposedBy: contributorGroup,
	}
}

func (p *PullRequestActivityProjections) apply(state *PullRequestActivityState, evt *ghevents.Event) {
	switch *evt.Payload.Action {
	case "opened":
		state.Opened++
	case "closed":
		if *evt.Payload.PullRequest.Merged {
			state.Merged++
		} else {
			state.ClosedWithUnmergedCommits++
		}
	}
}

func (p *PullRequestActivityProjections) applyMovingAverage(state *PullRequestActivityState, msg *PullRequestActivityState, windowSize int) {
	state.Opened += msg.Opened / float64(windowSize)
	state.Merged += msg.Merged / float64(windowSize)
	state.ClosedWithUnmergedCommits += msg.ClosedWithUnmergedCommits / float64(windowSize)
}

func (p *PullRequestActivityProjections) Register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
