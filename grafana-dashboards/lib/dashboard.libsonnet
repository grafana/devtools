local grafana = import 'grafonnet/grafana.libsonnet';
local dashboard = grafana.dashboard;
local link = grafana.link;
local annotation = grafana.annotation;
local template = import 'template.libsonnet';

{
  new(title, description, uid, config, releaseAnnotations=false) :: dashboard.new(
    title=title,
    description=description,
    schemaVersion=21,
    tags=['github-stats'],
    uid=uid,
    time_from=config.timeRange.from,
    time_to='now/w-1w',
    editable=true,
  )
  .addAnnotation(
    if releaseAnnotations then
      annotation.datasource(
        'Releases',
        config.datasource.name,
        iconColor='#64b0c8',
        enable=false,
      ) + config.datasource.queries.annotations.releases()
    else null)
  .addLink(link.dashboards(
    'GitHub Stats',
    ['github-stats'],
    includeVars=true,
  ))
}
