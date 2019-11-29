local githubstats = import 'lib/githubstats.libsonnet';
local sqlQueries = githubstats.sqlQueries;

function(dsType='mysql', dsName='github_stats', from='2014-12-31T23:00:00.000Z') {
  local config = {
    datasource: {
      type: dsType,
      name: dsName,
      queries: sqlQueries.new(dsType),
    },
    timeRange: {
      from: from,
    },
  },
  'issues_activity.json': githubstats.issuesActivity.new('gh-001', config),
  'pr_activity.json': githubstats.prActivity.new('gh-002', config),
}
