local grafana = import 'grafonnet/grafana.libsonnet';
local graph = grafana.graphPanel;

{
  new(title, description, datasource, aliasColors) ::
    graph.new(
      title,
      description=description,
      datasource=datasource,
      aliasColors=aliasColors,
      lines=false,
      bars=true,
      stack=true,
      decimals=2,
      legend_alignAsTable=true,
      legend_values=true,
      legend_min=true,
      legend_max=true,
      legend_current=true,
      legend_total=true,
      legend_avg=true,
      legend_sort='avg',
      legend_sortDesc=true,
      min=0,
      sort='decreasing',
    )
}
