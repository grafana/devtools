package githubstats

import (
	"time"

	ghevents "github.com/grafana/github-repo-metrics/pkg/streams"
	"github.com/grafana/github-repo-metrics/pkg/streams/pkg/streamprojections"
)

const (
	DailyEventsActivityStream                  = "DailyEventsActvity"
	WeeklyEventsActivityStream                 = "WeeklyEventsActvity"
	MonthlyEventsActivityStream                = "MonthlyEventsActvity"
	QuarterlyEventsActivityStream              = "QuarterlyEventsActvity"
	YearlyEventsActivityStream                 = "YearlyEventsActvity"
	SevenDaysMovingAverageEventsActivityStream = "SevenDaysMovingAverageEvents"
	EventsActivityStream                       = "EventsActivityStream"
)

type EventsActivityState struct {
	Time      time.Time `persist:",primarykey"`
	Period    string    `persist:",primarykey"`
	Repo      string    `persist:",primarykey"`
	EventType string    `persist:"event_type,primarykey"`
	Count     float64
}

type EventsActivityProjections struct {
	daily                  *streamprojections.StreamProjection
	weekly                 *streamprojections.StreamProjection
	monthly                *streamprojections.StreamProjection
	quarterly              *streamprojections.StreamProjection
	yearly                 *streamprojections.StreamProjection
	sevenDaysMovingAverage *streamprojections.StreamProjection
	all                    *streamprojections.StreamProjection
}

func NewEventsActivityProjections() *EventsActivityProjections {
	p := &EventsActivityProjections{}
	p.daily = streamprojections.
		FromStream(GithubEventStream).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(DailyEventsActivityStream).
		Build()

	p.weekly = streamprojections.
		FromStream(GithubEventStream).
		Weekly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(WeeklyEventsActivityStream).
		Build()

	p.monthly = streamprojections.
		FromStream(GithubEventStream).
		Monthly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(MonthlyEventsActivityStream).
		Build()

	p.quarterly = streamprojections.
		FromStream(GithubEventStream).
		Quarterly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(QuarterlyEventsActivityStream).
		Build()

	p.yearly = streamprojections.
		FromStream(GithubEventStream).
		Yearly(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		ToStream(YearlyEventsActivityStream).
		Build()

	p.sevenDaysMovingAverage = streamprojections.
		FromStream(GithubEventStream).
		Daily(fromCreatedDate, partitionByRepo, p.partitionByEventType).
		Init(p.init).
		Apply(p.apply).
		Window(6, 0, "d7", p.applyMovingAverage).
		ToStream(SevenDaysMovingAverageEventsActivityStream).
		Build()

	p.all = streamprojections.
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

func (p *EventsActivityProjections) init(t time.Time, repo, eventtype, period string) streamprojections.ProjectionState {
	return &EventsActivityState{Time: t, Period: period, Repo: repo, EventType: eventtype}
}

func (p *EventsActivityProjections) apply(state *EventsActivityState, evt *ghevents.Event) {
	state.Count++
}

func (p *EventsActivityProjections) applyMovingAverage(state *EventsActivityState, msg *EventsActivityState, windowSize int) {
	state.Count += msg.Count / float64(windowSize)
}

func (p *EventsActivityProjections) Register(engine streamprojections.StreamProjectionEngine) {
	engine.Register(p.daily)
	engine.Register(p.weekly)
	engine.Register(p.monthly)
	engine.Register(p.quarterly)
	engine.Register(p.yearly)
	engine.Register(p.sevenDaysMovingAverage)
	engine.Register(p.all)
}
