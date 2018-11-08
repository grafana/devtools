package archive

import (
	"log"
	"time"

	"github.com/go-xorm/xorm"
)

var (
	eventTypeIssue              = "IssuesEvent"
	eventTypeComment            = "IssueCommentEvent"
	watchEventType              = "WatchEvent"
	pushEventType               = "PushEvent"
	pullRequestEventType        = "PullRequestEvent"
	pullRequestCommentEventType = "PullRequestReviewCommentEvent"
	releaseEventType            = "ReleaseEvent"
	forkEventType               = "ForkEvent"
	memberEventType             = "MemberEvent"
	createEventType             = "CreateEvent"
	deleteEventType             = "DeleteEvent"
	commitCommentEvent          = "CommitCommentEvent"
)

// AggregatedStats is an aggregated view of
type AggregatedStats struct {
	ID                      int64
	IssueCount              int64
	IssueCommentCount       int64
	PrCount                 int64
	WatcherCount            int64
	PullRequestCommentCount int64
}

// Aggregator is responsible for aggregating stats based on github events
type Aggregator struct {
	engine    *xorm.Engine
	closeChan chan time.Time
}

// NewAggregator returns a new Aggregator
func NewAggregator(engine *xorm.Engine, closeChan chan time.Time) *Aggregator {
	return &Aggregator{engine: engine, closeChan: closeChan}
}

// Aggregate will read github events from the database and write back an aggregated view
func (a *Aggregator) Aggregate() error {
	events, err := a.findEvents()
	if err != nil {
		return err
	}

	aggs, err := a.aggregate(events)
	if err != nil {
		return err
	}

	for _, aggregation := range aggs {
		_, err := a.engine.Exec("DELETE FROM aggregated_stats WHERE id = ? ", aggregation.ID)
		if err != nil {
			return err
		}

		_, err = a.engine.Insert(aggregation)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Aggregator) aggregate(events []*GithubEvent) (map[int64]*AggregatedStats, error) {
	aggregations := map[int64]*AggregatedStats{}

	for _, e := range events {
		//id := time.Date(e.CreatedAt.Year(), e.CreatedAt.Month(), e.CreatedAt.Day(), e.CreatedAt.Hour(), 0, 0, 0, time.UTC).UTC().Unix()
		id := time.Date(e.CreatedAt.Year(), e.CreatedAt.Month(), e.CreatedAt.Day(), 0, 0, 0, 0, time.UTC).UTC().Unix()

		agg, exists := aggregations[id]
		if !exists {
			agg = &AggregatedStats{ID: id}
			aggregations[id] = agg
		}

		a.applyEvent(agg, e)
	}

	return aggregations, nil
}

func (a *Aggregator) applyEvent(agg *AggregatedStats, e *GithubEvent) {

	if e.Type == eventTypeIssue {
		action, err := e.Payload.Get("action").String()
		if err != nil {
			log.Fatalf("failed to get action. error: %+v\n", err)
		}

		if action != "" {
			switch action {
			case "opened":
				fallthrough
			case "reopened":
				agg.IssueCount++
			case "closed":
				agg.IssueCount--
			default:
				log.Printf("Unknown issue action: %s", action)
			}
		}
	}

	if e.Type == watchEventType {
		action, err := e.Payload.Get("action").String()
		if err != nil {
			log.Fatalf("failed to get action. error: %+v\n", err)
		}

		if action != "" {
			switch action {
			case "started":
				agg.WatcherCount++
			default:
				log.Fatalf("Unknown issue action: %s \n", action)
			}
		}
	}

	if e.Type == pullRequestCommentEventType {
		action, err := e.Payload.Get("action").String()
		if err != nil {
			log.Fatalf("failed to get action. error: %+v\n", err)
		}

		if action != "" {
			switch action {
			case "closed":
				agg.PullRequestCommentCount--
			case "created":
				agg.PullRequestCommentCount++
			default:
				log.Fatalln("unknown action: ", action)
			}
		}
	}

	if e.Type == pullRequestEventType {
		action, err := e.Payload.Get("action").String()
		if err != nil {
			log.Fatalf("failed to get action. error: %+v\n", err)
		}

		if action != "" {
			switch action {
			case "closed":
				agg.PrCount--
			case "reopened":
				fallthrough
			case "opened":
				agg.PrCount++
			default:
				log.Fatalln("unknown action: ", action)
			}
		}
	}

	if e.Type == eventTypeComment {
		action, err := e.Payload.Get("action").String()
		if err != nil {
			log.Fatalf("failed to get action. error: %+v\n", err)
		}

		if action != "" {
			switch action {
			case "created":
				agg.IssueCommentCount++
			default:
				log.Fatalln("unknown comment type: ", agg.IssueCommentCount)
			}
		}
	}
}

func (a *Aggregator) findEvents() ([]*GithubEvent, error) {
	result := []*GithubEvent{}

	rowCount := 0
	stepSize := 5000

	for {
		events := []*GithubEvent{}
		err := a.engine.Limit(stepSize, rowCount*stepSize).Find(&events)
		if err != nil {
			log.Fatalf("failed to query. error: %v", err)
		}

		if len(events) == 0 {
			return result, nil
		}

		result = append(result, events...)
		rowCount++
	}
}
