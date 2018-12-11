package githubstats

import (
	"reflect"
	"testing"
	"time"

	ghevents "github.com/grafana/devtools/pkg/ghevents"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUtils(t *testing.T) {
	Convey("Test utils", t, func() {
		Convey("fromCreatedDate", func() {
			now := time.Now()
			msg := &ghevents.Event{CreatedAt: now}
			sub := fromCreatedDate(reflect.ValueOf(msg).Interface())
			So(sub, ShouldEqual, now)
		})

		Convey("filterByOpenedAndClosedActions", func() {
			action := "opened"
			msg := &ghevents.Event{Payload: &ghevents.Payload{
				Action: &action,
			}}
			sub := filterByOpenedAndClosedActions(reflect.ValueOf(msg).Interface())
			So(sub, ShouldBeTrue)

			action = "closed"
			msg = &ghevents.Event{Payload: &ghevents.Payload{
				Action: &action,
			}}
			sub = filterByOpenedAndClosedActions(reflect.ValueOf(msg).Interface())
			So(sub, ShouldBeTrue)

			action = "other"
			msg = &ghevents.Event{Payload: &ghevents.Payload{
				Action: &action,
			}}
			sub = filterByOpenedAndClosedActions(reflect.ValueOf(msg).Interface())
			So(sub, ShouldBeFalse)
		})

		Convey("partitionByRepo", func() {
			msg := &ghevents.Event{Repo: &ghevents.Repo{ID: 1, Name: "test/repo"}}
			key, value := partitionByRepo(reflect.ValueOf(msg).Interface())
			So(key, ShouldEqual, "repo")
			So(value, ShouldEqual, "test/repo")
		})

		Convey("mapUserLoginToGroup", func() {
			userLoginGroupMap["user1"] = "group1"
			userLoginGroupMap["user2"] = "group2"
			So(mapUserLoginToGroup("user1"), ShouldEqual, "group1")
			So(mapUserLoginToGroup("user2"), ShouldEqual, "group2")
			So(mapUserLoginToGroup("user3"), ShouldEqual, "Contributor")
		})

		Convey("filterAndPatchRepos", func() {
			skipRepos = []string{"org/repo1", "org/repo2"}
			mapRepos = map[string]string{"org/repo-old": "org/repo-new"}
			msg := &ghevents.Event{Repo: &ghevents.Repo{ID: 1, Name: "org/repo1"}}
			So(filterAndPatchRepos(reflect.ValueOf(msg).Interface()), ShouldBeFalse)

			msg = &ghevents.Event{Repo: &ghevents.Repo{ID: 1, Name: "org/repo2"}}
			So(filterAndPatchRepos(reflect.ValueOf(msg).Interface()), ShouldBeFalse)

			msg = &ghevents.Event{Repo: &ghevents.Repo{ID: 1, Name: "org/repo3"}}
			So(filterAndPatchRepos(reflect.ValueOf(msg).Interface()), ShouldBeTrue)

			msg = &ghevents.Event{Repo: &ghevents.Repo{ID: 1, Name: "org/repo-old"}}
			So(filterAndPatchRepos(reflect.ValueOf(msg).Interface()), ShouldBeTrue)
			So(msg.Repo.Name, ShouldEqual, "org/repo-new")
		})

		Convey("isBot", func() {
			So(isBot("CLAassistant"), ShouldBeTrue)
			So(isBot("codecov-io"), ShouldBeTrue)
			So(isBot("user"), ShouldBeFalse)
		})

		Convey("Percentile", func() {
			values := []float64{43, 54, 56, 61, 62, 66, 68, 69, 69, 70, 71, 72, 77, 78, 79, 85, 87, 88, 89, 93, 95, 96, 98, 99, 99}

			Convey("p90", func() {
				p := percentile(0.9, values)
				So(p, ShouldEqual, 98)
			})

			Convey("p50", func() {
				p := percentile(0.5, values)
				So(p, ShouldEqual, 77)
			})

			Convey("p20", func() {
				p := percentile(0.2, values)
				So(p, ShouldEqual, 64)
			})
		})
	})
}
