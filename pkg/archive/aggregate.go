package archive

import (
	"log"
	"time"

	"github.com/go-xorm/xorm"
)

var (
	EventTypeIssue              = "IssuesEvent"
	EventTypeComment            = "IssueCommentEvent"
	WatchEvent                  = "WatchEvent"
	PushEventType               = "PushEvent"
	PullRequestEventType        = "PullRequestEvent"
	PullRequestCommentEventType = "PullRequestReviewCommentEvent"
	ReleaseEventType            = "ReleaseEvent"
	ForkEventType               = "ForkEvent"
	MemberEventType             = "MemberEvent"
	CreateEventType             = "CreateEvent"
	DeleteEventType             = "DeleteEvent"
	CommitCommentEvent          = "CommitCommentEvent"
	//GollumEventType             = "GollumEvent"
)

type AggregatedStats struct {
	ID                      int64
	IssueCount              int64
	IssueCommentCount       int64
	PrCount                 int64
	WatcherCount            int64
	PullRequestCommentCount int64
}

type Aggregator struct {
	engine    *xorm.Engine
	closeChan chan time.Time
}

func NewAggregator(engine *xorm.Engine, closeChan chan time.Time) *Aggregator {
	return &Aggregator{engine: engine, closeChan: closeChan}
}

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
		id := time.Date(e.CreatedAt.Year(), e.CreatedAt.Month(), e.CreatedAt.Day(), e.CreatedAt.Hour(), 0, 0, 0, time.UTC).UTC().Unix()

		agg, exists := aggregations[id]
		if !exists {
			agg = &AggregatedStats{ID: id}
			aggregations[id] = agg
		}

		if e.Type == EventTypeIssue {
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

		if e.Type == WatchEvent {
			action, err := e.Payload.Get("action").String()
			if err != nil {
				log.Fatalf("failed to get action. error: %+v\n", err)
			}

			if action != "" {
				switch action {
				case "started":
					agg.WatcherCount++
				default:
					log.Printf("Unknown issue action: %s", action)
				}
			}
		}

		if e.Type == PullRequestCommentEventType {
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

		if e.Type == PullRequestEventType {
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

		if e.Type == EventTypeComment {
			action, err := e.Payload.Get("action").String()
			if err != nil {
				log.Fatalf("failed to get action. error: %+v\n", err)
			}

			if action != "" {
				switch action {
				case "created":
					agg.IssueCommentCount++
				default:
					log.Println("unknown comment type: ", agg.IssueCommentCount)
				}
			}
		}
	}

	return aggregations, nil
}

func (a *Aggregator) findEvents() ([]*GithubEvent, error) {
	events := []*GithubEvent{}

	err := a.engine.Limit(100000, 0).Find(&events)
	if err != nil {
		log.Fatalf("failed to query. error: %v", err)
	}

	return events, nil
}
