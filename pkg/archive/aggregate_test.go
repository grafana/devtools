package archive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

func TestAggregateIssueCount(t *testing.T) {
	jsonIssueOpened, err := simplejson.NewJson([]byte(`{ "action": "opened" }`))
	assert.Nil(t, err, "json opened ")
	jsonIssueClosed, err := simplejson.NewJson([]byte(`{ "action": "closed" }`))
	assert.Nil(t, err, "json closed ")

	dayOne := time.Date(2015, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
	dayTwo := time.Date(2015, time.Month(1), 2, 0, 0, 0, 0, time.UTC)
	dayThree := time.Date(2015, time.Month(1), 3, 0, 0, 0, 0, time.UTC)

	tc := []struct {
		Aggregations map[int64]*AggregatedStats
		Events       []*GithubEvent
	}{
		{
			Aggregations: map[int64]*AggregatedStats{
				dayOne.UTC().Unix():   &AggregatedStats{IssueCount: 1},
				dayTwo.UTC().Unix():   &AggregatedStats{IssueCount: 3},
				dayThree.UTC().Unix(): &AggregatedStats{IssueCount: 1},
			},
			Events: []*GithubEvent{
				mockGithubEvent(EventTypeIssue, dayOne.Add(time.Hour).UTC(), jsonIssueOpened),

				mockGithubEvent(EventTypeIssue, dayTwo.Add(time.Hour).UTC(), jsonIssueOpened),
				mockGithubEvent(EventTypeIssue, dayTwo.Add(time.Hour).UTC(), jsonIssueOpened),
				mockGithubEvent(EventTypeIssue, dayTwo.Add(time.Hour).UTC(), jsonIssueOpened),
				mockGithubEvent(EventTypeIssue, dayTwo.Add(time.Hour).UTC(), jsonIssueOpened),
				mockGithubEvent(EventTypeIssue, dayTwo.Add(time.Hour).UTC(), jsonIssueClosed),

				mockGithubEvent(EventTypeIssue, dayThree.Add(time.Hour).UTC(), jsonIssueOpened),
			},
		},
	}

	ag := NewAggregator(nil, make(chan time.Time))
	for _, test := range tc {
		aggs, err := ag.aggregate(test.Events)
		assert.Nil(t, err, "aggreagte should not return error")

		for date, testAggregation := range test.Aggregations {
			aggregation, exists := aggs[date]
			if exists {
				assert.Equal(t, aggregation.IssueCount, testAggregation.IssueCount)
			} else {
				assert.Fail(t, "could not find aggregation state")
			}
		}
	}
}

func mockGithubEvent(typ string, timestamp time.Time, paylaod *simplejson.Json) *GithubEvent {
	ge := &GithubEvent{
		ID:      timestamp.UTC().Unix(),
		Payload: paylaod,
		Type:    typ,
	}

	ge.CreatedAt = timestamp.UTC()

	return ge
}
