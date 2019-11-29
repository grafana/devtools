{
  releases() :: {
    rawQuery: "SELECT
  \"time\",
  \"title\" AS text,
  \"tags\"
FROM
  \"release_annotation\"
WHERE
  $__unixEpochFilter(\"time\") AND
  \"repo\" = 'grafana/grafana' AND
  \"prerelease\" = false"
  }
}
