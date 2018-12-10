package githubstats

import (
	"time"

	ghevents "github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
)

const issuesViewStream = "IssuesView"

type issuesViewState struct {
	id       int
	repo     string
	openedBy string
	openedAt time.Time
	closedAt time.Time
	closed   bool
}

type issuesViewProjections struct {
	view *streamprojections.StreamProjection
}

func newIssuesViewProjections() *issuesViewProjections {
	p := &issuesViewProjections{}
	p.view = streamprojections.
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

func (p *issuesViewProjections) init(id int, repo string) streamprojections.ProjectionState {
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

func (p *issuesViewProjections) register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.view)
}

const (
	DailyIssuesAgeStream                  = "DailyIssuesAge"
	WeeklyIssuesAgeStream                 = "WeeklyIssuesAge"
	MonthlyIssuesAgeStream                = "MonthlyIssuesAge"
	QuarterlyIssuesAgeStream              = "QuarterlyIssuesAge"
	YearlyIssuesAgeStream                 = "YearlyIssuesAge"
	IssuesAgeStream                       = "IssuesAgeStream"
	SevenDaysMovingAverageIssuesAgeStream = "SevenDaysMovingAverageIssuesAge"
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
	daily                  *streamprojections.StreamProjection
	sevenDaysMovingAverage *streamprojections.StreamProjection
	weekly                 *streamprojections.StreamProjection
	monthly                *streamprojections.StreamProjection
	quarterly              *streamprojections.StreamProjection
	yearly                 *streamprojections.StreamProjection
	all                    *streamprojections.StreamProjection
}

func NewIssuesAgeProjections() *IssuesAgeProjections {
	p := &IssuesAgeProjections{}

	p.ViewProjections = newIssuesViewProjections()

	p.daily = streamprojections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(DailyIssuesAgeStream).
		Build()

	p.weekly = streamprojections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Weekly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(WeeklyIssuesAgeStream).
		Build()

	p.monthly = streamprojections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Monthly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(MonthlyIssuesAgeStream).
		Build()

	p.quarterly = streamprojections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Quarterly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(QuarterlyIssuesAgeStream).
		Build()

	p.yearly = streamprojections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Yearly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(YearlyIssuesAgeStream).
		Build()

	p.sevenDaysMovingAverage = streamprojections.
		FromStream(issuesViewStream).
		Filter(p.filterByClosed).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageIssuesAgeStream).
		Build()

	p.all = streamprojections.
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

func (p *IssuesAgeProjections) init(t time.Time, repo, openedBy, period string) streamprojections.ProjectionState {
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

func (p *IssuesAgeProjections) done(stateArr []streamprojections.ProjectionState) {
	for _, s := range stateArr {
		ageState := s.(*IssuesAgeState)
		ageState.MedianAge = percentile(0.5, ageState.ageItems)
	}
}

func (p *IssuesAgeProjections) applyMovingAverage(state *IssuesAgeState, msg *IssuesAgeState, windowSize int) {
	state.MedianAge += msg.MedianAge / float64(windowSize)
}

func (p *IssuesAgeProjections) Register(engine streamprojections.StreamProjectionEngine) {
	p.ViewProjections.register(engine)
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
