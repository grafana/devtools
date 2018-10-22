#/bin/sh

docker run  \
    --rm \
    --net=host \
    -e "STATS_DB=postgres" \
    -e "STATS_CONNSTRING=postgresql://grafana:password@localhost:5432/grafana?sslmode=disable" \
    -e "STATS_MAXDURATION=10" \
    grafana/github-stats-aggregate:latest

