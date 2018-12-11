package githubstats

import (
	"github.com/grafana/devtools/pkg/streams/projections"
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

func RegisterProjections(pe projections.StreamProjectionEngine) {
	NewSplitByEventTypeProjections().Register(pe)
	NewIssuesActivityProjections().Register(pe)
	NewPullRequestActivityProjections().Register(pe)
	NewIssueCommentsActivityProjections().Register(pe)
	NewPullRequestCommentsActivityProjections().Register(pe)
	NewEventsActivityProjections().Register(pe)
	NewReleaseAnnotationProjections().Register(pe)
	NewCommitActivityProjections().Register(pe)
	NewForksActivityProjections().Register(pe)
	NewStargazersActivityProjections().Register(pe)
	NewPullRequestAgeProjections().Register(pe)
	NewPullRequestOpenedToMergedProjections().Register(pe)
	NewIssuesAgeProjections().Register(pe)

	for _, login := range githubLogins {
		userLoginGroupMap[login] = "Grafana Labs"
	}

	skipRepos = []string{
		"grafana/ansible-grafana",
		"grafana/appdynamics-grafana-datasource",
		"grafana/app-plugin-example",
		"grafana/azure-marketplace",
		"grafana/cortex",
		"grafana/datasource-plugin-generic",
		"grafana/datasource-plugin-genericdatasource",
		"grafana/devtools",
		"grafana/dockerana",
		"grafana/docs-base",
		"grafana/example-app",
		"grafana/fake-data-gen",
		"grafana/gemini-scrollbar",
		"grafana/github-repo-metrics",
		"grafana/github-to-es",
		"grafana/globalconf",
		"grafana/grafana-api-golang-client",
		"grafana/grafana-api-nodejs-client",
		"grafana/grafana-cli",
		"grafana/grafana-docker-dev-env",
		"grafana/grafana.org",
		"grafana/grafana-packer",
		"grafana/grafana_plugin_model",
		"grafana/grafana-plugin-model",
		"grafana/grafana-plugins",
		"grafana/grafana-tools",
		"grafana/grafana-wixtoolset",
		"grafana/gridstack.js",
		"grafana/heatmap-panel",
		"grafana/homebrew-core",
		"grafana/homebrew-grafana",
		"grafana/hosted-metrics-sender-example",
		"grafana/mt-qa",
		"grafana/plugin_model",
		"grafana/puppet-grafana",
		"grafana/react-grid-layout",
		"grafana/snap_k8s",
		"grafana/snap-plugin-collector-cadvisor",
		"grafana/snap-plugin-collector-gitstats",
		"grafana/snap-plugin-collector-kubestate",
		"grafana/starter-panel",
		"grafana/synthesize",
		"grafana/usage-stats-handler",
		"grafana/vertica-datasource",
	}

	mapRepos = map[string]string{
		"grafana/":                              "grafana/grafana",
		"grafana/grafana-docker":                "grafana/grafana",
		"grafana/grafana-build-container":       "grafana/grafana",
		"grafana/azure-kusto-datasource":        "grafana/azure-data-explorer-datasource",
		"grafana/datasource-plugin-influxdb-08": "grafana/influxdb-08-datasource",
		"grafana/datasource-plugin-kairosdb":    "grafana/kairosdb-datasource",
		"grafana/panel-plugin-piechart":         "grafana/piechart-panel",
	}
}
