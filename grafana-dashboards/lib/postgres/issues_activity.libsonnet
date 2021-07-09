local sqlQuery = import '../sql_query.libsonnet';

{
  time: {
    byPeriodRepoAndOpenedBy(config) :: sqlQuery.timeseries(
"SELECT
  $__time(time),
  sum(opened) AS \"Opened\",
  sum(closed) AS \"Closed\"
FROM
  issues_activity
WHERE
  $__unixEpochFilter(time) AND
  period = '${period}' AND
  repo IN(${repo}) AND
  opened_by IN(${opened_by})
GROUP BY 1
ORDER BY 1",
      datasource=config.datasource.name,
    ),

    openedByPeriodAndRepo(config) :: sqlQuery.timeseries(
"SELECT
  $__time(time),
  opened_by,
  sum(opened)
FROM
  issues_activity
WHERE
  $__unixEpochFilter(time) AND
  period = '${period}' AND
  repo IN(${repo})
GROUP BY 1,2
ORDER BY 1,2",
      datasource=config.datasource.name,
    ),

    closedByPeriodAndRepo(config) :: sqlQuery.timeseries(
"SELECT
  $__time(time),
  opened_by,
  sum(closed)
FROM
  issues_activity
WHERE
  $__unixEpochFilter(time) AND
  period = '${period}' AND
  repo IN(${repo})
GROUP BY 1,2
ORDER BY 1,2",
      datasource=config.datasource.name,
    )
  },

  openedBy() :: "SELECT DISTINCT opened_by FROM issues_activity",
}
