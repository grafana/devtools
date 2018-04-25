package bots

import (
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// Export GITSTATS_ACCESS_TOKEN as an env variable to run the tests
func TestFeatureRequestBot(t *testing.T) {
	if os.Getenv("GITSTATS_ACCESS_TOKEN") == "" {
		return
	}

	Convey("Get All Stale Feature Requests from Grafana", t, func() {
		token := os.Getenv("GITSTATS_ACCESS_TOKEN")
		bot := NewFeatureRequestBot(token, "grafana", "grafana")
		issues, err := bot.GetStaleFeatureRequests()

		So(err, ShouldBeNil)
		for _, issue := range issues {
			fmt.Println()
			fmt.Printf("%v;%v;%v;%v;https://github.com/grafana/grafana/issues/%v;", *issue.Number, *issue.Title, *issue.State, *issue.CreatedAt, *issue.Number)
		}
		So(len(issues), ShouldBeGreaterThan, 0)
	})
}
