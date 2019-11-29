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
    title='Pull Request Activity',
    description='Pull request activity (proposed, merged, closed with unmerged commits) over time in total and grouped by which group proposed the the pull request',
    releaseAnnotations=true
  ) :: dashboard.new(
    title=title,
    description=description,
    uid=uid,
    config=config,
    releaseAnnotations=releaseAnnotations,
  )
  .addTemplates([
    template.period(config, 'pr_activity', 'w'),
    template.repo(config, 'pr_activity'),
    template.proposedBy(config),
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
    $.panels.proposedBarGraph(config),
    gridPos={
      h: 11,
      w: 24,
      x: 0,
      y: 11
    }
  )
  .addPanel(
    $.panels.mergedBarGraph(config),
    gridPos={
      h: 11,
      w: 24,
      x: 0,
      y: 22,
    }
  )
  .addPanel(
    $.panels.closedWithUnmergedCommitsBarGraph(config),
    gridPos={
      h: 11,
      w: 24,
      x: 0,
      y: 33,
    }
  ),

  panels: {
    activityBarGraph(
      config,
      title='Pull Request Activity ($period)',
      description='Proposed and merged count of pull requests over time. See [definition](https://github.com/grafana/devtools/blob/master/pkg/githubstats/pr_activity.go)',
      aliasColors={
        'Merged': '#6f42c1',
        'Proposed': '#28a745'
      },
    ) :: barGraph.new(
      title=title,
      description=description,
      datasource=config.datasource.name,
      aliasColors=aliasColors,
    )
    .addTarget(
      config.datasource.queries.prActivity.time.byPeriodRepoAndProposedBy(config),
    ),

    proposedBarGraph(
      config,
      title='Proposed Pull Requests By ($period)',
      description='Proposed (opened) pull requests by group, over time. See [definition](https://github.com/grafana/devtools/blob/master/pkg/githubstats/pr_activity.go)',
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
      config.datasource.queries.prActivity.time.proposedByPeriodAndRepo(config),
    ),

    mergedBarGraph(
      config,
      title='Merged Pull Requests Proposed By ($period)',
      description='Merged pull requests proposed by group, over time. See [definition](https://github.com/grafana/devtools/blob/master/pkg/githubstats/pr_activity.go)',
      aliasColors={
        "Contributor": "rgb(75, 31, 156)",
        "Grafana Labs": "#6f42c1"
      },
    ) :: barGraph.new(
      title=title,
      description=description,
      datasource=config.datasource.name,
      aliasColors=aliasColors,
    )
    .addTarget(
      config.datasource.queries.prActivity.time.mergedByPeriodAndRepo(config),
    ),

    closedWithUnmergedCommitsBarGraph(
      config,
      title='Closed Pull Request With Unmerged Commits Proposed By ($period)',
      description='Closed pull requests with unmerged commits proposed by group, over time. See [definition](https://github.com/grafana/devtools/blob/master/pkg/githubstats/pr_activity.go)',
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
      config.datasource.queries.prActivity.time.closedWithUnmergedCommitsByPeriodAndRepo(config),
    ),
  },
}