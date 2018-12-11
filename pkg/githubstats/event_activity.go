package githubstats

import (
	"time"

	"github.com/grafana/devtools/pkg/ghevents"
	"github.com/grafana/devtools/pkg/streams/projections"
)

const (
	DailyEventsActivityStream                  = "d_events"
	WeeklyEventsActivityStream                 = "w_events"
	MonthlyEventsActivityStream                = "m_events"
	QuarterlyEventsActivityStream              = "q_events"
	YearlyEventsActivityStream                 = "y_events"
	SevenDaysMovingAverageEventsActivityStream = "d7_events"
	EventsActivityStream                       = "all_events"
)

type EventsActivityState struct {
	Time      time.Time `persist:",primarykey"`
	Period    string    `persist:",primarykey"`
	Repo      string    `persist:",primarykey"`
	EventType string    `persist:"event_type,primarykey"`
	Count     float64
}

type EventsActivityProjections struct {
	daily                  *projections.StreamProjection
	weekly                 *projections.StreamProjection
	monthly                *projections.StreamProjection
	quarterly              *projections.StreamProjection
	yearly                 *projections.StreamProjection
	sevenDaysMovingAverage *projections.StreamProjection
	all                    *projections.StreamProjection
}

func NewEventsActivityProjections() *EventsActivityProjections {
	p := &EventsActivityProjections{}
	p.daily = projections.
		FromStream(GithubEventStream).
		Filter(filterAndPatchRepos).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyEventsActivityStream).
		Build()

	p.weekly = projections.
		FromStream(GithubEventStream).
		Filter(filterAndPatchRepos).
		Weekly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyEventsActivityStream).
		Build()

	p.monthly = projections.
		FromStream(GithubEventStream).
		Filter(filterAndPatchRepos).
		Monthly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyEventsActivityStream).
		Build()

	p.quarterly = projections.
		FromStream(GithubEventStream).
		Filter(filterAndPatchRepos).
		Quarterly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyEventsActivityStream).
		Build()

	p.yearly = projections.
		FromStream(GithubEventStream).
		Filter(filterAndPatchRepos).
		Yearly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyEventsActivityStream).
		Build()

	p.sevenDaysMovingAverage = projections.
		FromStream(GithubEventStream).
		Filter(filterAndPatchRepos).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageEventsActivityStream).
		Build()

	p.all = projections.
		FromStreams(
			DailyEventsActivityStream,
			WeeklyEventsActivityStream,
			MonthlyEventsActivityStream,
			QuarterlyEventsActivityStream,
			YearlyEventsActivityStream,
			SevenDaysMovingAverageEventsActivityStream,
		).
		ToStream(EventsActivityStream).
		Persist("events_activity", &EventsActivityState{}).
		Build()

	return p
}

func (p *EventsActivityProjections) partitionByEventType(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "eventType", evt.Type
}

func (p *EventsActivityProjections) init(t time.Time, repo, eventtype, period string) projections.ProjectionState {
	return &EventsActivityState{Time: t, Period: period, Repo: repo, EventType: eventtype}
}

func (p *EventsActivityProjections) apply(state *EventsActivityState, evt *ghevents.Event) {
	state.Count++
}

func (p *EventsActivityProjections) applyMovingAverage(state *EventsActivityState, msg *EventsActivityState, windowSize int) {
	state.Count += msg.Count / float64(windowSize)
}

func (p *EventsActivityProjections) Register(engine projections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
