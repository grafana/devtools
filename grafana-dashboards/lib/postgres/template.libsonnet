{
  periods(tableName) :: "SELECT
  t.__value,
  t.__text
FROM (
  SELECT DISTINCT
    period as __value,
    case period
      when 'h24' then '24 hours moving average'
      when 'd' then 'Daily'
      when 'd7' then '7 days moving average'
      when 'w' then 'Weekly'
      when 'm' then 'Monthly'
      when 'q' then 'Quarterly'
      when 'y' then 'Yearly'
    end as __text,
    case period
      when 'h24' then 1
      when 'd' then 2
      when 'd7' then 3
      when 'w' then 4
      when 'm' then 5
      when 'q' then 6
      when 'y' then 7
    end as order
  FROM
    " + tableName + "
) as t
ORDER BY
  t.order",

repos(tableName) :: "SELECT DISTINCT repo FROM " + tableName,

}