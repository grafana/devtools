# Github repo metrics

A bundle of CLI applications for gathering/extracting data from github 
and turning it into actionable data.

download arch files and serve locally
```bash
cd test-data
wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
go run main.go
```

commands:
```bash
curl -I https://api.github.com/repos/grafana/grafana/events

docker run -it --rm jbergknoff/postgresql-client postgresql://user:pass@host:5432/db

docker run -it --rm jbergknoff/postgresql-client postgresql://githubstats:githubstats@localhost:5432/githubstats
```
