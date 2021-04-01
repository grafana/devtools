package githubstats

import (
	"github.com/grafana/devtools/pkg/streams/projections"
)

var githubLogins = []string{
	"09jvilla",
	"56quarters",
	"achatterjee-grafana",
	"adeverteuil",
	"aengusrooneygrafana",
	"afreels95",
	"AgnesToulet",
	"ahadjidj",
	"aknuds1",
	"alexanderzobnin",
	"alexishensel",
	"alicefarrell",
	"aligerrard",
	"annanay25",
	"annelaurefroment",
	"aocenas",
	"aSquare14",
	"atifali",
	"beorn7",
	"bergquist",
	"besartberisha",
	"bfmatei",
	"bojankezele",
	"bossinc",
	"briangann",
	"brittenhouse",
	"buro9",
	"camsellem",
	"captncraig",
	"cchomp",
	"chaudyg",
	"chieubpham",
	"ChrisDGH",
	"christina-thaute",
	"christinagillman",
	"christineywang",
	"cinaglia",
	"Clarity-89",
	"clord",
	"codesome",
	"colega",
	"cristiangreco",
	"csmarchbanks",
	"cstyan",
	"csw3190",
	"cyriltovena",
	"dafydd-t",
	"DanCech",
	"daniellee",
	"danielpalay",
	"danjensen206",
	"danmdinu",
	"dannykopping",
	"darrenjaneczek",
	"davidmparrott",
	"davkal",
	"dconrique01",
	"ddorman16",
	"DeeGeeGit",
	"dfrankel33",
	"Dieterbe",
	"domasx2",
	"douglashanna",
	"dprokop",
	"dsotirakis",
	"Duologic",
	"eamonryan",
	"EduSpencer",
	"electron0zero",
	"Elfo404",
	"ethirolle",
	"eyalonGL",
	"fionaliao",
	"fkaleo",
	"FZambia",
	"Gabfana",
	"gabor",
	"gamab",
	"gassiss",
	"gerboland",
	"GinaBerlin",
	"gotjosh",
	"gouthamve",
	"gracesharpe",
	"grafana-eddie",
	"grafana-support",
	"grafanabot",
	"Graham-Moreno",
	"hairyhenderson",
	"hcave-sf",
	"hedss",
	"hemdrup",
	"hjet",
	"hoenn",
	"hugohaggmark",
	"ifrost",
	"Ijin08",
	"inf0rmer",
	"inkel",
	"isaackkim",
	"ivanahuckova",
	"jackw",
	"jdbaldry",
	"jeremy-watkins",
	"jeschkies",
	"jessabe",
	"jessover9000",
	"jesswyrick",
	"jesusvazquez",
	"jgrossman1",
	"jhsilver2",
	"jhunthrop",
	"jimnagy",
	"jkirkwood",
	"jmarbach",
	"joanlopez",
	"joe-elliott",
	"johnheywardobrien",
	"JohnLee-Grafana",
	"jongyllen",
	"joshhunt",
	"josiahg",
	"jtlisi",
	"julienduchesne",
	"jvrplmlmn",
	"katrina-odfina",
	"kavirajk",
	"kaydelaney",
	"kminehart",
	"kmiroy",
	"Ktsuewright",
	"kylebrandt",
	"l-m-gamble",
	"leeoniya",
	"leventebalogh",
	"lizzyleahy",
	"lukasztyrala",
	"lux4rd0",
	"macgreagoir",
	"malcolmholmes",
	"mapno",
	"marcusolsson",
	"marefr",
	"mattdurham",
	"matthewhelmke",
	"mattmendick",
	"mattttt",
	"mccollam",
	"mckn",
	"mderynck",
	"mdisibio",
	"melgl",
	"mellieA",
	"mem",
	"MichelHollands",
	"mikempx",
	"mjseaman",
	"mlewald",
	"mplzik",
	"muaazsaleem",
	"myrle-krantz",
	"natellium",
	"nathanrodman",
	"nchristus",
	"nikoalch",
	"nikosmeds",
	"nopzor1200",
	"oanamangiurea",
	"oddlittlebird",
	"oscarkilhed",
	"osg-grafana",
	"owen-d",
	"papagian",
	"peterholmberg",
	"pracucci",
	"pstibrany",
	"qkombur",
	"radiohead",
	"rdubrock",
	"replay",
	"rfratto",
	"rgeyer",
	"richardqlam",
	"RichiH",
	"robbymilo",
	"robert-milan",
	"rpsondhi",
	"ryantxu",
	"Rydez",
	"s-h-a-d-o-w",
	"sakjur",
	"samjewell",
	"sandeepsukhani",
	"sarlinska",
	"scoren-gl",
	"scottlepp",
	"sd2k",
	"sdboyer",
	"sfingerhut",
	"sh0rez",
	"simonc6372",
	"simonswine",
	"slim-bean",
	"songyhan",
	"srclosson",
	"stevesg",
	"suitupalex",
	"sunker",
	"teddyba",
	"thisisobate",
	"thomson",
	"Tomica-G",
	"tomwilkie",
	"torkelo",
	"trevorwhitney",
	"trotttrotttrott",
	"undef1nd",
	"VesnaDean",
	"vickyyyyyyy",
	"vincent-grafana",
	"vtolani",
	"vtorosyan",
	"wardbekker",
	"wbrowne",
	"woodsaj",
	"ximsandthings",
	"xlson",
	"yep-grafana",
	"yesoreyeram",
	"ying-jeanne",
	"zoltanbedi",
	"zuchka",
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
