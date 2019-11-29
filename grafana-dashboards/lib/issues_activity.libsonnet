local githubstats = import 'githubstats.libsonnet';
local dashboard = githubstats.dashboard;
local template = githubstats.template;
local annotation = githubstats.annotation;
local link = githubstats.link;
local barGraph = githubstats.barGraphPanel;
local sqlQuery = import 'sql_query.libsonnet';

{
  new(
    uid,
    config,
    title='Issues Activity',
    description='Issues activity (opened, closed) over time period',
    releaseAnnotations=true
  ) :: dashboard.new(
    title=title,
    description=description,
    uid=uid,
    config=config,
    releaseAnnotations=releaseAnnotations,
  )
  .addTemplates([
    template.period(config, 'issues_activity', 'w'),
    template.repo(config, 'issues_activity'),
    template.openedBy(config),
  ])
  .addPanel(
    $.panels.activityBarGraph(config),
    gridPos={
      x: 0,
      y: 0,
      w: 24,
      h: 11,
    }
  )
  .addPanel(
    $.panels.openedBarGraph(config),
    gridPos={
      h: 11,
      w: 24,
      x: 0,
      y: 11,
    }
  )
  .addPanel(
    $.panels.closedBarGraph(config),
    gridPos={
      h: 11,
      w: 24,
      x: 0,
      y: 22,
    }
  ),

  panels: {
    activityBarGraph(
      config,
      title='Issues Activity ($period)',
      description='Opened and closed issues over time. See [definition](https://github.com/grafana/devtools/blob/master/pkg/githubstats/issues_activity.go)',
      aliasColors={
        'Opened': 'light-green',
        'Closed': 'light-red',
      },
    ) :: barGraph.new(
      title=title,
      description=description,
      datasource=config.datasource.name,
      aliasColors=aliasColors,
    )
    .addTarget(
      config.datasource.queries.issuesActivity.time.byPeriodRepoAndOpenedBy(config),
    ),

    openedBarGraph(
      config,
      title='Opened Issues By ($period)',
      description='Opened issues by author group over time. See [definition](https://github.com/grafana/devtools/blob/master/pkg/githubstats/issues_activity.go)',
      aliasColors={
        "Contributor": "rgb(24, 92, 40)",
        "Grafana Labs": "#28a745",
      },
    ) :: barGraph.new(
      title=title,
      description=description,
      datasource=config.datasource.name,
      aliasColors=aliasColors,
    )
    .addTarget(
      config.datasource.queries.issuesActivity.time.openedByPeriodAndRepo(config)
    ),

    closedBarGraph(
      config,
      title='Closed Issues Opened By ($period). See [definition](https://github.com/grafana/devtools/blob/master/pkg/githubstats/issues_activity.go)',
      description='Closed issues opened by author group over time',
      datasource=config.datasource.name,
      aliasColors={
        "Contributor": "rgb(152, 44, 55)",
        "Grafana Labs": "#d73a49",
      },
    ) :: barGraph.new(
      title=title,
      description=description,
      datasource=config.datasource.name,
      aliasColors=aliasColors,
    )
    .addTarget(
      config.datasource.queries.issuesActivity.time.closedByPeriodAndRepo(config)
    ),
  }
}