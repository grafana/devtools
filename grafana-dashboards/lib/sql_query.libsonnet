{
  timeseries(
    rawSql,
    datasource=null,
  ):: {
    [if datasource != null then 'datasource']: datasource,
    format: 'time_series',
    rawQuery: true,
    rawSql: rawSql,
  },
  table(
    rawSql,
    datasource=null,
  ):: {
    [if datasource != null then 'datasource']: datasource,
    format: 'table',
    rawQuery: true,
    rawSql: rawSql,
  },
}
