package githubstats

import (
	"math"
	"sort"
	"time"

	ghevents "github.com/grafana/github-repo-metrics/pkg/streams"
)

const coreContributorGroupName = "Core Contributor"
const contributorGroupName = "Contributor"

func fromCreatedDate(msg interface{}) time.Time {
	evt := msg.(*ghevents.Event)
	return evt.CreatedAt
}

func filterByOpenedAndClosedActions(msg interface{}) bool {
	evt := msg.(*ghevents.Event)
	return *evt.Payload.Action == "opened" || *evt.Payload.Action == "closed"
}

func partitionByRepo(msg interface{}) (string, interface{}) {
	evt := msg.(*ghevents.Event)
	return "repo", evt.Repo.Name
}

var actorContributorGroupMap = map[string]string{
	"alexanderzobnin": coreContributorGroupName,
	"bergquist":       coreContributorGroupName,
	"briangann":       coreContributorGroupName,
	"bulletfactory":   coreContributorGroupName,
	"cstyan":          coreContributorGroupName,
	"DanCech":         coreContributorGroupName,
	"daniellee":       coreContributorGroupName,
	"davkal":          coreContributorGroupName,
	"Dieterbe":        coreContributorGroupName,
	"gouthamve":       coreContributorGroupName,
	"Ijin08":          coreContributorGroupName,
	"jimnagy":         coreContributorGroupName,
	"jschill":         coreContributorGroupName,
	"jtlisi":          coreContributorGroupName,
	"malcolmholmes":   coreContributorGroupName,
	"marefr":          coreContributorGroupName,
	"mattttt":         coreContributorGroupName,
	"nikoalch":        coreContributorGroupName,
	"nopzor1200":      coreContributorGroupName,
	"peterholmberg":   coreContributorGroupName,
	"replay":          coreContributorGroupName,
	"tomwilkie":       coreContributorGroupName,
	"torkelo":         coreContributorGroupName,
	"woodsaj":         coreContributorGroupName,
	"xlson":           coreContributorGroupName,
}

func percentile(k float64, values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	index := k * float64(len(values))

	if index != float64(int64(index)) {
		index = math.Round(index)
		if int(index) == 0 {
			return values[int(index)]
		}
		return values[int(index)-1]
	}

	slice := values[int(index)-1 : int(index)+1]
	return (slice[0] + slice[1]) / 2
}
