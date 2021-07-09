{
  new(dsType) ::
    if dsType == 'postgres' then
      import 'postgres/queries.libsonnet'
    else
      import 'mysql/queries.libsonnet'
}