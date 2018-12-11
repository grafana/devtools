package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	DailyPullRequestCommentsActivityStream                  = "d_pr_comments"
	WeeklyPullRequestCommentsActivityStream                 = "w_pr_comments"
	MonthlyPullRequestCommentsActivityStream                = "m_pr_comments"
	QuarterlyPullRequestCommentsActivityStream              = "q_pr_comments"
	YearlyPullRequestCommentsActivityStream                 = "y_pr_comments"
	SevenDaysMovingAveragePullRequestCommentsActivityStream = "d7_pr_comments"
	PullRequestCommentsActivityStream                       = "all_pr_comments"
)

type PullRequestCommentsActivityState struct {
	Time       time.Time `persist:",primarykey"`
	Period     string    `persist:",primarykey"`
	Repo       string    `persist:",primarykey"`
	AuthoredBy string    `persist:"authored_by,primarykey"`
	Count      float64
}

type PullRequestCommentsActivityProjections struct {
	daily                  *projections.StreamProjection
	sevenDaysMovingAverage *projections.StreamProjection
	weekly                 *projections.StreamProjection
	monthly                *projections.StreamProjection
	quarterly              *projections.StreamProjection
	yearly                 *projections.StreamProjection
	all                    *projections.StreamProjection
}

func NewPullRequestCommentsActivityProjections() *PullRequestCommentsActivityProjections {
	p := &PullRequestCommentsActivityProjections{}
	p.daily = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterPullRequestComments).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyPullRequestCommentsActivityStream).
		Build()

	p.weekly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterPullRequestComments).
		Weekly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyPullRequestCommentsActivityStream).
		Build()

	p.monthly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterPullRequestComments).
		Monthly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyPullRequestCommentsActivityStream).
		Build()

	p.quarterly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterPullRequestComments).
		Quarterly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyPullRequestCommentsActivityStream).
		Build()

	p.yearly = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterPullRequestComments).
		Yearly(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyPullRequestCommentsActivityStream).
		Build()

	p.sevenDaysMovingAverage = projections.
		FromStream(IssueCommentEventStream).
		Filter(p.filterPullRequestComments).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByAuthoredBy).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAveragePullRequestCommentsActivityStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyPullRequestCommentsActivityStream,
			WeeklyPullRequestCommentsActivityStream,
			MonthlyPullRequestCommentsActivityStream,
			QuarterlyPullRequestCommentsActivityStream,
			YearlyPullRequestCommentsActivityStream,
			SevenDaysMovingAveragePullRequestCommentsActivityStream,
		).
		ToStream(PullRequestCommentsActivityStream).
		Persist("pr_comments_activity", &PullRequestCommentsActivityState{}).
		Build()

	return p
}

func (p *PullRequestCommentsActivityProjections) filterPullRequestComments(msg interface{}) bool {
	evt := msg.(*ghevents.Event)

	return *evt.Payload.Action == "created" && evt.Payload.Issue.PullRequest != nil && !isBot(evt.Actor.Login)
}

func (p *PullRequestCommentsActivityProjections) partitionByAuthoredBy(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "authoredBy", mapUserLoginToGroup(evt.Actor.Login)
}

func (p *PullRequestCommentsActivityProjections) init(t time.Time, repo, contributorGroup, period string) projections.ProjectionState {
	return &PullRequestCommentsActivityState{
		Time:       t,
		Period:     period,
		Repo:       repo,
		AuthoredBy: contributorGroup,
	}
}

func (p *PullRequestCommentsActivityProjections) apply(state *PullRequestCommentsActivityState, evt *ghevents.Event) {
	state.Count++
}

func (p *PullRequestCommentsActivityProjections) applyMovingAverage(state *PullRequestCommentsActivityState, msg *PullRequestCommentsActivityState, windowSize int) {
	state.Count += msg.Count / float64(windowSize)
}

func (p *PullRequestCommentsActivityProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
