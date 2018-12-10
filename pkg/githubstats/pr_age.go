package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const pullRequestViewStream = "PullRequestView"

type pullRequestViewState struct {
	id       int
	repo     string
	openedBy string
	openedAt time.Time
	closedAt time.Time
	closed   bool
	merged   bool
}

type pullRequestViewProjections struct {
	view *projections.StreamProjection
}

func newPullRequestViewProjections() *pullRequestViewProjections {
	p := &pullRequestViewProjections{}
	p.view = projections.
		FromStream(PullRequestEventStream).
		PartitionBy(p.partitionByPrID, partitionByRepo).
		Init(p.init).
		Apply(p.apply).
		ToStream(pullRequestViewStream).
		Build()

	return p
}

func (p *pullRequestViewProjections) partitionByPrID(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "id", evt.Payload.PullRequest.ID
}

func (p *pullRequestViewProjections) init(id int, repo string) projections.ProjectionState {
	return &pullRequestViewState{
		id:   id,
		repo: repo,
	}
}

func (p *pullRequestViewProjections) apply(state *pullRequestViewState, evt *ghevents.Event) {
	switch *evt.Payload.Action {
	case "opened":
		state.openedAt = evt.CreatedAt
		state.openedBy = evt.Actor.Login
	case "closed":
		state.closed = true
		state.closedAt = evt.CreatedAt
		state.merged = evt.Payload.PullRequest.Merged != nil && *evt.Payload.PullRequest.Merged
	case "reopened":
		state.closed = false
	}
}

func (p *pullRequestViewProjections) register(engine projections.StreamProjectionEngine) {
	engine.Register(p.view)
}

const (
	DailyPullRequestAgeStream                  = "DailyPullRequestAge"
	WeeklyPullRequestAgeStream                 = "WeeklyPullRequestAge"
	MonthlyPullRequestAgeStream                = "MonthlyPullRequestAge"
	QuarterlyPullRequestAgeStream              = "QuarterlyPullRequestAge"
	YearlyPullRequestAgeStream                 = "YearlyPullRequestAge"
	PullRequestAgeStream                       = "PullRequestAgeStream"
	SevenDaysMovingAveragePullRequestAgeStream = "SevenDaysMovingAveragePullRequestAge"
)

type PullRequestAgeState struct {
	Time       time.Time `persist:",primarykey"`
	Period     string    `persist:",primarykey"`
	Repo       string    `persist:",primarykey"`
	ProposedBy string    `persist:"proposed_by,primarykey"`
	MedianAge  float64   `persist:"median_age"`
	ageItems   []float64
}

type PullRequestAgeProjections struct {
	ViewProjections        *pullRequestViewProjections
	daily                  *projections.StreamProjection
	sevenDaysMovingAverage *projections.StreamProjection
	weekly                 *projections.StreamProjection
	monthly                *projections.StreamProjection
	quarterly              *projections.StreamProjection
	yearly                 *projections.StreamProjection
	all                    *projections.StreamProjection
}

func NewPullRequestAgeProjections() *PullRequestAgeProjections {
	p := &PullRequestAgeProjections{}

	p.ViewProjections = newPullRequestViewProjections()

	p.daily = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByOpened).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(DailyPullRequestAgeStream).
		Build()

	p.weekly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByOpened).
		Weekly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(WeeklyPullRequestAgeStream).
		Build()

	p.monthly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByOpened).
		Monthly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(MonthlyPullRequestAgeStream).
		Build()

	p.quarterly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByOpened).
		Quarterly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(QuarterlyPullRequestAgeStream).
		Build()

	p.yearly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByOpened).
		Yearly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(YearlyPullRequestAgeStream).
		Build()

	p.sevenDaysMovingAverage = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByOpened).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAveragePullRequestAgeStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyPullRequestAgeStream,
			WeeklyPullRequestAgeStream,
			MonthlyPullRequestAgeStream,
			QuarterlyPullRequestAgeStream,
			YearlyPullRequestAgeStream,
			SevenDaysMovingAveragePullRequestAgeStream,
		).
		ToStream(PullRequestAgeStream).
		Persist("pr_age", &PullRequestAgeState{}).
		Build()

	return p
}

func (p *PullRequestAgeProjections) filterByOpened(msg interface{}) bool {
	state := msg.(*pullRequestViewState)
	return !state.openedAt.IsZero()
}

func (p *PullRequestAgeProjections) partitionByClosedAt(msg interface{}) time.Time {
	state := msg.(*pullRequestViewState)
	return state.openedAt
}

func (p *PullRequestAgeProjections) partitionByRepo(msg interface{}) (string, interface{}) {
	state := msg.(*pullRequestViewState)
	return "repo", state.repo
}

func (p *PullRequestAgeProjections) partitionByProposedBy(msg interface{}) (string, interface{}) {
	state := msg.(*pullRequestViewState)
	return "proposedBy", mapUserLoginToGroup(state.openedBy)
}

func (p *PullRequestAgeProjections) init(t time.Time, repo, proposedBy, period string) projections.ProjectionState {
	return &PullRequestAgeState{
		Time:       t,
		Period:     period,
		Repo:       repo,
		ProposedBy: proposedBy,
		ageItems:   []float64{},
	}
}

func (p *PullRequestAgeProjections) apply(state *PullRequestAgeState, viewState *pullRequestViewState) {
	if !viewState.closed {
		viewState.closedAt = time.Now().UTC()
	}
	state.ageItems = append(state.ageItems, viewState.closedAt.Sub(viewState.openedAt).Seconds())
}

func (p *PullRequestAgeProjections) done(stateArr []projections.ProjectionState) {
	for _, s := range stateArr {
		ageState := s.(*PullRequestAgeState)
		ageState.MedianAge = percentile(0.5, ageState.ageItems)
	}
}

func (p *PullRequestAgeProjections) applyMovingAverage(state *PullRequestAgeState, msg *PullRequestAgeState, windowSize int) {
	state.MedianAge += msg.MedianAge / float64(windowSize)
}

func (p *PullRequestAgeProjections) Register(engine projections.StreamProjectionEngine) {
	p.ViewProjections.register(engine)
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}

const (
	DailyPullRequestOpenedToMergedStream                  = "DailyPullRequestOpenedToMerged"
	WeeklyPullRequestOpenedToMergedStream                 = "WeeklyPullRequestOpenedToMerged"
	MonthlyPullRequestOpenedToMergedStream                = "MonthlyPullRequestOpenedToMerged"
	QuarterlyPullRequestOpenedToMergedStream              = "QuarterlyPullRequestOpenedToMerged"
	YearlyPullRequestOpenedToMergedStream                 = "YearlyPullRequestOpenedToMerged"
	PullRequestOpenedToMergedStream                       = "PullRequestOpenedToMergedStream"
	SevenDaysMovingAveragePullRequestOpenedToMergedStream = "SevenDaysMovingAveragePullRequestOpenedToMerged"
)

type PullRequestOpenedToMergedState struct {
	Time         time.Time `persist:",primarykey"`
	Period       string    `persist:",primarykey"`
	Repo         string    `persist:",primarykey"`
	ProposedBy   string    `persist:"proposed_by,primarykey"`
	Percentile15 float64   `persist:"p15"`
	Percentile50 float64   `persist:"p50"`
	Percentile85 float64   `persist:"p85"`
	ageItems     []float64
}

type PullRequestOpenedToMergedProjections struct {
	ViewProjections        *pullRequestViewProjections
	daily                  *projections.StreamProjection
	sevenDaysMovingAverage *projections.StreamProjection
	weekly                 *projections.StreamProjection
	monthly                *projections.StreamProjection
	quarterly              *projections.StreamProjection
	yearly                 *projections.StreamProjection
	all                    *projections.StreamProjection
}

func NewPullRequestOpenedToMergedProjections() *PullRequestOpenedToMergedProjections {
	p := &PullRequestOpenedToMergedProjections{}

	p.daily = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByMerged).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(DailyPullRequestOpenedToMergedStream).
		Build()

	p.weekly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByMerged).
		Weekly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(WeeklyPullRequestOpenedToMergedStream).
		Build()

	p.monthly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByMerged).
		Monthly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(MonthlyPullRequestOpenedToMergedStream).
		Build()

	p.quarterly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByMerged).
		Quarterly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(QuarterlyPullRequestOpenedToMergedStream).
		Build()

	p.yearly = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByMerged).
		Yearly(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		ToStream(YearlyPullRequestOpenedToMergedStream).
		Build()

	p.sevenDaysMovingAverage = projections.
		FromStream(pullRequestViewStream).
		Filter(p.filterByMerged).
		Daily(p.partitionByClosedAt, p.partitionByRepo, p.partitionByProposedBy).
		Init(p.init).
		Apply(p.apply).
		Done(p.done).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAveragePullRequestOpenedToMergedStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyPullRequestOpenedToMergedStream,
			WeeklyPullRequestOpenedToMergedStream,
			MonthlyPullRequestOpenedToMergedStream,
			QuarterlyPullRequestOpenedToMergedStream,
			YearlyPullRequestOpenedToMergedStream,
			SevenDaysMovingAveragePullRequestOpenedToMergedStream,
		).
		ToStream(PullRequestOpenedToMergedStream).
		Persist("pr_opened_to_merged", &PullRequestOpenedToMergedState{}).
		Build()

	return p
}

func (p *PullRequestOpenedToMergedProjections) filterByMerged(msg interface{}) bool {
	state := msg.(*pullRequestViewState)
	return !state.openedAt.IsZero() && state.merged
}

func (p *PullRequestOpenedToMergedProjections) partitionByClosedAt(msg interface{}) time.Time {
	state := msg.(*pullRequestViewState)
	return state.openedAt
}

func (p *PullRequestOpenedToMergedProjections) partitionByRepo(msg interface{}) (string, interface{}) {
	state := msg.(*pullRequestViewState)
	return "repo", state.repo
}

func (p *PullRequestOpenedToMergedProjections) partitionByProposedBy(msg interface{}) (string, interface{}) {
	state := msg.(*pullRequestViewState)
	return "proposedBy", mapUserLoginToGroup(state.openedBy)
}

func (p *PullRequestOpenedToMergedProjections) init(t time.Time, repo, proposedBy, period string) projections.ProjectionState {
	return &PullRequestOpenedToMergedState{
		Time:       t,
		Period:     period,
		Repo:       repo,
		ProposedBy: proposedBy,
		ageItems:   []float64{},
	}
}

func (p *PullRequestOpenedToMergedProjections) apply(state *PullRequestOpenedToMergedState, viewState *pullRequestViewState) {
	state.ageItems = append(state.ageItems, viewState.closedAt.Sub(viewState.openedAt).Seconds())
}

func (p *PullRequestOpenedToMergedProjections) done(stateArr []projections.ProjectionState) {
	for _, s := range stateArr {
		ageState := s.(*PullRequestOpenedToMergedState)
		ageState.Percentile15 = percentile(0.15, ageState.ageItems)
		ageState.Percentile50 = percentile(0.5, ageState.ageItems)
		ageState.Percentile85 = percentile(0.85, ageState.ageItems)
	}
}

func (p *PullRequestOpenedToMergedProjections) applyMovingAverage(state *PullRequestOpenedToMergedState, msg *PullRequestOpenedToMergedState, windowSize int) {
	state.Percentile15 += msg.Percentile15 / float64(windowSize)
	state.Percentile50 += msg.Percentile50 / float64(windowSize)
	state.Percentile85 += msg.Percentile85 / float64(windowSize)
}

func (p *PullRequestOpenedToMergedProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
