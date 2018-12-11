package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	DailyIssueCommentsActivityStream                  = "DailyIssueCommentsActvity"
	WeeklyIssueCommentsActivityStream                 = "WeeklyIssueCommentsActvity"
	MonthlyIssueCommentsActivityStream                = "MonthlyIssueCommentsActvity"
	QuarterlyIssueCommentsActivityStream              = "QuarterlyIssueCommentsActvity"
	YearlyIssueCommentsActivityStream                 = "YearlyIssueCommentsActvity"
	IssueCommentsActivityStream                       = "IssueCommentsActivityStream"
	SevenDaysMovingAverageIssueCommentsActivityStream = "SevenDaysMovingAverageIssueCommentsActivity"
)

type IssueCommentsActivityState struct {
	Time       time.Time `persist:",primarykey"`
	Period     string    `persist:",primarykey"`
	Repo       string    `persist:",primarykey"`
	AuthoredBy string    `persist:"authored_by,primarykey"`
	Count      float64
}

type IssueCommentsActivityProjections struct {
	daily                  *projections.StreamProjection
	sevenDaysMovingAverage *projections.StreamProjection
	weekly                 *projections.StreamProjection
	monthly                *projections.StreamProjection
	quarterly              *projections.StreamProjection
	yearly                 *projections.StreamProjection
	all                    *projections.StreamProjection
}

func NewIssueCommentsActivityProjections() *IssueCommentsActivityProjections {
	p := &IssueCommentsActivityProjections{}
	p.daily = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterIssueComments).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyIssueCommentsActivityStream).
		Build()

	p.weekly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterIssueComments).
		Weekly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyIssueCommentsActivityStream).
		Build()

	p.monthly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterIssueComments).
		Monthly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyIssueCommentsActivityStream).
		Build()

	p.quarterly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterIssueComments).
		Quarterly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyIssueCommentsActivityStream).
		Build()

	p.yearly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterIssueComments).
		Yearly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyIssueCommentsActivityStream).
		Build()

	p.sevenDaysMovingAverage = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterIssueComments).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageIssueCommentsActivityStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyIssueCommentsActivityStream,
			WeeklyIssueCommentsActivityStream,
			MonthlyIssueCommentsActivityStream,
			QuarterlyIssueCommentsActivityStream,
			YearlyIssueCommentsActivityStream,
			SevenDaysMovingAverageIssueCommentsActivityStream,
		).
		ToStream(IssueCommentsActivityStream).
		Persist("issue_comments_activity", &IssueCommentsActivityState{}).
		Build()

	return p
}

func (p *IssueCommentsActivityProjections) filterIssueComments(msg interface{}) bool {
	evt := msg.(*ghevents.Event)
	return *evt.Payload.Action == "created" && evt.Payload.Issue.PullRequest == nil && !isBot(evt.Actor.Login)
}

func (p *IssueCommentsActivityProjections) partitionByAuthoredBy(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "authoredBy", mapUserLoginToGroup(evt.Actor.Login)
}

func (p *IssueCommentsActivityProjections) init(t time.Time, repo, contributorGroup, period string) projections.ProjectionState {
	return &IssueCommentsActivityState{
		Time:       t,
		Period:     period,
		Repo:       repo,
		AuthoredBy: contributorGroup,
	}
}

func (p *IssueCommentsActivityProjections) apply(state *IssueCommentsActivityState, evt *ghevents.Event) {
	state.Count++
}

func (p *IssueCommentsActivityProjections) applyMovingAverage(state *IssueCommentsActivityState, msg *IssueCommentsActivityState, windowSize int) {
	state.Count += msg.Count / float64(windowSize)
}

func (p *IssueCommentsActivityProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
