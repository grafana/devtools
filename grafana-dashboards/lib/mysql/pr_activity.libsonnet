local sqlQuery = import '../sql_query.libsonnet';

{
  time: {
    byPeriodRepoAndProposedBy(config) :: sqlQuery.timeseries(
"SELECT
  time,
  sum(opened) AS \"Proposed\",
  sum(merged) AS \"Merged\"
FROM
  pr_activity
WHERE
  $__unixEpochFilter(time) AND
  period = '${period}' AND
  repo IN(${repo}) AND
  proposed_by IN(${proposed_by})
GROUP BY 1
ORDER BY 1",
      datasource=config.datasource.name,
    ),

    proposedByPeriodAndRepo(config) :: sqlQuery.timeseries(
"SELECT
  time,
  proposed_by,
  sum(opened)
FROM
  pr_activity
WHERE
  $__unixEpochFilter(time) AND
  period = '${period}'AND
  repo IN(${repo})
GROUP BY 1,2
ORDER BY 1,2",
      datasource=config.datasource.name,
    ),

    mergedByPeriodAndRepo(config) :: sqlQuery.timeseries(
"SELECT
  time,
  proposed_by,
  sum(merged)
FROM
  pr_activity
WHERE
  $__unixEpochFilter(time) AND
  period = '${period}' AND
  repo IN(${repo})
GROUP BY 1,2
ORDER BY 1,2",
      datasource=config.datasource.name,
    ),

    closedWithUnmergedCommitsByPeriodAndRepo(config) :: sqlQuery.timeseries(
"SELECT
  time,
  proposed_by
  sum(closed_with_unmerged_commits)
FROM
  pr_activity
WHERE
  $__unixEpochFilter(time) AND
  period = '${period}' AND
  repo IN(${repo}) AND
GROUP BY 1,2
ORDER BY 1,2",
      datasource=config.datasource.name,
    ),

  },

  proposedBy() :: "SELECT DISTINCT proposed_by FROM pr_activity",
}
