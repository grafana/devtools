package githubstats

import (
	"time"

	ghevents "github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
)

const (
	DailyIssuesActivityStream                  = "DailyIssuesActvity"
	WeeklyIssuesActivityStream                 = "WeeklyIssuesActvity"
	MonthlyIssuesActivityStream                = "MonthlyIssuesActvity"
	QuarterlyIssuesActivityStream              = "QuarterlyIssuesActvity"
	YearlyIssuesActivityStream                 = "YearlyIssuesActvity"
	IssuesActivityStream                       = "IssuesActivityStream"
	SevenDaysMovingAverageIssuesActivityStream = "SevenDaysMovingAverageIssuesActivity"
)

type IssuesActivityState struct {
	Time     time.Time `persist:",primarykey"`
	Period   string    `persist:",primarykey"`
	Repo     string    `persist:",primarykey"`
	OpenedBy string    `persist:"opened_by,primarykey"`
	Opened   float64
	Closed   float64
}

type IssuesActivityProjections struct {
	daily                  *streamprojections.StreamProjection
	sevenDaysMovingAverage *streamprojections.StreamProjection
	weekly                 *streamprojections.StreamProjection
	monthly                *streamprojections.StreamProjection
	quarterly              *streamprojections.StreamProjection
	yearly                 *streamprojections.StreamProjection
	all                    *streamprojections.StreamProjection
}

func NewIssuesActivityProjections() *IssuesActivityProjections {
	p := &IssuesActivityProjections{}
	p.daily = streamprojections.
		FromStream(IssuesEventStream).
		Filter(filterByOpenedAndClosedActions).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByIssueAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyIssuesActivityStream).
		Build()

	p.weekly = streamprojections.
		FromStream(IssuesEventStream).
		Filter(filterByOpenedAndClosedActions).
		Weekly(fromCreatedDate, partitionByRepo, p.partitionByIssueAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyIssuesActivityStream).
		Build()

	p.monthly = streamprojections.
		FromStream(IssuesEventStream).
		Filter(filterByOpenedAndClosedActions).
		Monthly(fromCreatedDate, partitionByRepo, p.partitionByIssueAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyIssuesActivityStream).
		Build()

	p.quarterly = streamprojections.
		FromStream(IssuesEventStream).
		Filter(filterByOpenedAndClosedActions).
		Quarterly(fromCreatedDate, partitionByRepo, p.partitionByIssueAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyIssuesActivityStream).
		Build()

	p.yearly = streamprojections.
		FromStream(IssuesEventStream).
		Filter(filterByOpenedAndClosedActions).
		Yearly(fromCreatedDate, partitionByRepo, p.partitionByIssueAuthor).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyIssuesActivityStream).
		Build()

	p.sevenDaysMovingAverage = streamprojections.
		FromStream(IssuesEventStream).
		Filter(filterByOpenedAndClosedActions).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByIssueAuthor).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageIssuesActivityStream).
		Build()

	p.all = streamprojections.
		FromStreams(
			DailyIssuesActivityStream,
			WeeklyIssuesActivityStream,
			MonthlyIssuesActivityStream,
			QuarterlyIssuesActivityStream,
			YearlyIssuesActivityStream,
			SevenDaysMovingAverageIssuesActivityStream,
		).
		ToStream(IssuesActivityStream).
		Persist("issues_activity", &IssuesActivityState{}).
		Build()

	return p
}

func (p *IssuesActivityProjections) partitionByIssueAuthor(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)

	contributorGroup := contributorGroupName

	if group, ok := actorContributorGroupMap[evt.Payload.Issue.User.Login]; ok {
		contributorGroup = group
	}

	return "openedBy", contributorGroup
}

func (p *IssuesActivityProjections) init(t time.Time, repo, contributorGroup, period string) streamprojections.ProjectionState {
	return &IssuesActivityState{
		Time:     t,
		Period:   period,
		Repo:     repo,
		OpenedBy: contributorGroup,
	}
}

func (p *IssuesActivityProjections) apply(state *IssuesActivityState, evt *ghevents.Event) {
	switch *evt.Payload.Action {
	case "opened":
		state.Opened++
	case "closed":
		state.Closed++
	}
}

func (p *IssuesActivityProjections) applyMovingAverage(state *IssuesActivityState, msg *IssuesActivityState, windowSize int) {
	state.Opened += msg.Opened / float64(windowSize)
	state.Closed += msg.Closed / float64(windowSize)
}

func (p *IssuesActivityProjections) Register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
