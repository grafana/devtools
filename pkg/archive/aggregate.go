package archive

import (
	"log"
	"time"

	"github.com/go-xorm/xorm"
)

type AggregatedStats struct {
	ID                int64
	IssueCount        int64
	IssueCommentCount int64
	PrCount           int64
	WatcherCount      int64
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

var (
	EventTypeIssue   = "IssuesEvent"
	EventTypeComment = "IssueCommentEvent"
	WatchEvent       = "WatchEvent"
)

func (a *Aggregator) aggregate(events []*GithubEvent) (map[int64]*AggregatedStats, error) {
	aggregations := map[int64]*AggregatedStats{}

	var issueCount int64 = 0
	var issueCommentCount int64 = 0
	var prCount int64 = 0
	var watcherCount int64 = 0

	for _, e := range events {
		id := time.Date(e.CreatedAt.Year(), e.CreatedAt.Month(), e.CreatedAt.Day(), 0, 0, 0, 0, time.UTC).UTC().Unix()

		if _, exists := aggregations[id]; !exists {
			aggregations[id] = &AggregatedStats{
				ID:           id,
				IssueCount:   issueCount,
				PrCount:      prCount,
				WatcherCount: watcherCount,
			}
		}

		if e.Type == EventTypeIssue {
			issueStat, err := e.Payload.Get("action").String()
			if err == nil && issueStat != "" {
				switch issueStat {
				case "opened":
					issueCount++
				case "closed":
					issueCount--
				default:
					log.Printf("Unknown issue action: %s", issueStat)
				}
			}
		}

		if e.Type == WatchEvent {
			issueStat, err := e.Payload.Get("action").String()
			if err == nil && issueStat != "" {
				switch issueStat {
				case "started":
					watcherCount++
				//case "closed":
				//	issueCount--
				default:
					log.Printf("Unknown issue action: %s", issueStat)
				}
			}
		}

		if e.Type == EventTypeComment {
			issueCommentCount++
		}

		aggregations[id].IssueCount = issueCount
		aggregations[id].IssueCommentCount = issueCommentCount
		aggregations[id].PrCount = prCount
		aggregations[id].WatcherCount = watcherCount
	}

	return aggregations, nil
}

func (a *Aggregator) findEvents() ([]*GithubEvent, error) {
	events := []*GithubEvent{}

	err := a.engine.Limit(1000, 0).Find(&events)
	if err != nil {
		log.Fatalf("failed to query. error: %v", err)
	}

	return events, nil
}
