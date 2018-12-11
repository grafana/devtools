package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const issuesViewStream = "issues_view"

type issuesViewState struct {
	id       int
	repo     string
	openedBy string
	openedAt time.Time
	closedAt time.Time
	closed   bool
}

type issuesViewProjections struct {
	view *projections.StreamProjection
}

func newIssuesViewProjections() *issuesViewProjections {
	p := &issuesViewProjections{}
	p.view = projections.
		FromStream(IssuesEventStream).
		PartitionBy(p.partitionByIssueID, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(issuesViewStream).
		Build()

	return p
}

func (p *issuesViewProjections) partitionByIssueID(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "id", evt.Payload.Issue.ID
}

func (p *issuesViewProjections) init(id int, repo string) projections.ProjectionState {
	return &issuesViewState{
		id:   id,
		repo: repo,
	}
}

func (p *issuesViewProjections) apply(state *issuesViewState, evt *ghevents.Event) {
	switch *evt.Payload.Action {
	case "opened":
		state.openedAt = evt.CreatedAt
		state.openedBy = evt.Actor.Login
	case "closed":
		state.closed = true
		state.closedAt = evt.CreatedAt
	case "reopened":
		state.closed = false
	}
}

func (p *issuesViewProjections) register(engine projections.StreamProjectionEngine) {
	engine.Register(p.view)
}

const (
	DailyIssuesAgeStream                  = "d_issues_age"
	WeeklyIssuesAgeStream                 = "w_issues_age"
	MonthlyIssuesAgeStream                = "m_issues_age"
	QuarterlyIssuesAgeStream              = "q_issues_age"
	YearlyIssuesAgeStream                 = "y_issues_age"
	SevenDaysMovingAverageIssuesAgeStream = "d7_issues_age"
	IssuesAgeStream                       = "all_issues_age"
)

type IssuesAgeState struct {
	Time      time.Time `persist:",primarykey"`
	Period    string    `persist:",primarykey"`
	Repo      string    `persist:",primarykey"`
	OpenedBy  string    `persist:"opened_by,primarykey"`
	MedianAge float64   `persist:"median_age"`
	ageItems  []float64
}

type IssuesAgeProjections struct {
	ViewProjections        *issuesViewProjections
	daily                  *projections.StreamProjection
	sevenDaysMovingAverage *projections.StreamProjection
	weekly                 *projections.StreamProjection
	monthly                *projections.StreamProjection
	quarterly              *projections.StreamProjection
	yearly                 *projections.StreamProjection
	all                    *projections.StreamProjection
}

func NewIssuesAgeProjections() *IssuesAgeProjections {
	p := &IssuesAgeProjections{}

	p.ViewProjections = newIssuesViewProjections()

	p.daily = projections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(DailyIssuesAgeStream).
		Build()

	p.weekly = projections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Weekly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(WeeklyIssuesAgeStream).
		Build()

	p.monthly = projections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Monthly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(MonthlyIssuesAgeStream).
		Build()

	p.quarterly = projections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Quarterly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(QuarterlyIssuesAgeStream).
		Build()

	p.yearly = projections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Yearly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(YearlyIssuesAgeStream).
		Build()

	p.sevenDaysMovingAverage = projections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageIssuesAgeStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyIssuesAgeStream,
			WeeklyIssuesAgeStream,
			MonthlyIssuesAgeStream,
			QuarterlyIssuesAgeStream,
			YearlyIssuesAgeStream,
			SevenDaysMovingAverageIssuesAgeStream,
		).
		ToStream(IssuesAgeStream).
		Persist("issues_age", &IssuesAgeState{}).
		Build()

	return p
}

func (p *IssuesAgeProjections) filterByClosed(msg interface{}) bool {
	state := msg.(*issuesViewState)
	return state.closed && !state.openedAt.IsZero()
}

func (p *IssuesAgeProjections) partitionByClosedAt(msg interface{}) time.Time {
	state := msg.(*issuesViewState)
	return state.closedAt
}

func (p *IssuesAgeProjections) partitionByRepo(msg interface{}) (string, interface{}) {
	state := msg.(*issuesViewState)
	return "repo", state.repo
}

func (p *IssuesAgeProjections) partitionByProposedBy(msg interface{}) (string, interface{}) {
	state := msg.(*issuesViewState)
	return "openedBy", mapUserLoginToGroup(state.openedBy)
}

func (p *IssuesAgeProjections) init(t time.Time, repo, openedBy, period string) projections.ProjectionState {
	return &IssuesAgeState{
		Time:     t,
		Period:   period,
		Repo:     repo,
		OpenedBy: openedBy,
		ageItems: []float64{},
	}
}

func (p *IssuesAgeProjections) apply(state *IssuesAgeState, viewState *issuesViewState) {
	state.ageItems = append(state.ageItems, viewState.closedAt.Sub(viewState.openedAt).Seconds())
}

func (p *IssuesAgeProjections) done(stateArr []projections.ProjectionState) {
	for _, s := range stateArr {
		ageState := s.(*IssuesAgeState)
		ageState.MedianAge = percentile(0.5, ageState.ageItems)
	}
}

func (p *IssuesAgeProjections) applyMovingAverage(state *IssuesAgeState, msg *IssuesAgeState, windowSize int) {
	state.MedianAge += msg.MedianAge / float64(windowSize)
}

func (p *IssuesAgeProjections) Register(engine projections.StreamProjectionEngine) {
	p.ViewProjections.register(engine)
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
