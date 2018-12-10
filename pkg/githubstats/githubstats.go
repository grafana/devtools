package githubstats

import (
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
)

var githubLogins = []string{
	"adeverteuil",
	"alexanderzobnin",
	"bergquist",
	"briangann",
	"bulletfactory",
	"codesome",
	"cstyan",
	"DanCech",
	"daniellee",
	"davkal",
	"Dieterbe",
	"gouthamve",
	"Ijin08",
	"jimnagy",
	"jschill",
	"jtlisi",
	"malcolmholmes",
	"marefr",
	"mattttt",
	"nikoalch",
	"nopzor1200",
	"peterholmberg",
	"replay",
	"RichiH",
	"robert-milan",
	"tomwilkie",
	"torkelo",
	"woodsaj",
	"xlson",
}

func RegisterProjections(pe streamprojections.StreamProjectionEngine) {
	for _, login := range githubLogins {
		userLoginGroupMap[login] = "Grafana Labs"
	}

	NewSplitByEventTypeProjections().Register(pe)
	NewIssuesActivityProjections().Register(pe)
	NewPullRequestActivityProjections().Register(pe)
	NewEventsActivityProjections().Register(pe)
	NewReleaseAnnotationProjections().Register(pe)
	NewCommitActivityProjections().Register(pe)
	NewForksActivityProjections().Register(pe)
	NewStargazersActivityProjections().Register(pe)
	NewPullRequestAgeProjections().Register(pe)
	NewPullRequestOpenedToMergedProjections().Register(pe)
	NewIssuesAgeProjections().Register(pe)
	NewIssuesOpenClosedActivityProjections().Register(pe)
}
