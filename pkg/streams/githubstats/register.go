package githubstats

import (
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
)

func RegisterProjections(projectionEngine streamprojections.StreamProjectionEngine) {
	NewSplitByEventTypeProjections().Register(projectionEngine)
	NewIssuesActivityProjections().Register(projectionEngine)
	NewPullRequestActivityProjections().Register(projectionEngine)
	NewEventsActivityProjections().Register(projectionEngine)
	NewReleaseAnnotationProjections().Register(projectionEngine)
	NewCommitActivityProjections().Register(projectionEngine)
	NewForksActivityProjections().Register(projectionEngine)
	NewStargazersActivityProjections().Register(projectionEngine)
	NewPullRequestAgeProjections().Register(projectionEngine)
	NewPullRequestOpenedToMergedProjections().Register(projectionEngine)
	NewIssuesAgeProjections().Register(projectionEngine)
	NewIssuesOpenClosedActivityProjections().Register(projectionEngine)
}
