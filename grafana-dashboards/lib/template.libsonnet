local grafana = import 'grafonnet/grafana.libsonnet';
local template = grafana.template;

local periods = {
  'h24': '24 hours moving average',
  'd': 'Daily',
  'd7': '7 days moving average',
  'w': 'Weekly',
  'm': 'Monthly',
  'q': 'Quarterly',
  'y': 'Yearly',
};

{
  period(config, tableName, current='w') :: template.new(
      'period',
      datasource=config.datasource.name,
      query=config.datasource.queries.template.periods(tableName),
    label='Period',
    current=current,
  ),

  repo(config, tableName) :: template.new(
    'repo',
    datasource=config.datasource.name,
    query=config.datasource.queries.template.repos(tableName),
    label='Repository',
    current='grafana/grafana',
    includeAll=true,
    multi=true,
  ),

  openedBy(config) :: template.new(
    'opened_by',
    datasource=config.datasource.name,
    query=config.datasource.queries.issuesActivity.openedBy(),
    label='Opened By',
    includeAll=true,
    multi=true,
  ),

  proposedBy(config) :: template.new(
    'proposed_by',
    datasource=config.datasource.name,
    query=config.datasource.queries.prActivity.proposedBy(),
    label='Proposed By',
    includeAll=true,
    multi=true,
  ),
}
