#/bin/sh

docker run  \
    --net=host \
    -e "STATS_DB=postgres" \
    -e "STATS_CONNSTRING=postgresql://grafana:password@localhost:5432/grafana?sslmode=disable" \
    -e "STATS_MAXDURATION=10" \
    -e "STATS_ARCHIVEURL=http://data.githubarchive.org/%d-%02d-%02d-%d.json.gz" \
    -e "STATS_STARTDATEFLAG=2015-01-01" \
    -e "STATS_STOPDATEFLAG=" \
    grafana/github-stats-archive:latest

