package githubclient

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGitClient(t *testing.T) {
	if os.Getenv("GITSTATS_ACCESS_TOKEN") == "" {
		return
	}

	Convey("Get All Issues", t, func() {
		token := os.Getenv("GITSTATS_ACCESS_TOKEN")
		client := NewClient(token)
		issues, err := client.GetAllPullRequestsForMonth(context.Background(), "grafana", "grafana", time.May)

		So(err, ShouldBeNil)
		for _, issue := range issues {
			fmt.Printf("%v;%v;%v;%v;https://github.com/grafana/grafana/issues/%v;", *issue.Number, *issue.Title, *issue.State, *issue.CreatedAt, *issue.Number)
			fmt.Println()
		}
		So(len(issues), ShouldBeGreaterThan, 0)
	})
}
