# goals with this repo?

- Find new contributors.
  - follow up flagged contributors
  - see if flagged contributors keep providing.
- define SLA / SLO
- find issues/PR's without a response.
- open, closed issues
- open, closed PR's
- including/excluding core team.

# TODO
- download all events from github archive
- filter out event for Grafana
- store each event in postgre
- process events from postgre


download arch files
```bash
cd archive
wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
```

commands:
```bash
curl -I https://api.github.com/repos/grafana/grafana/events

docker run -it --rm jbergknoff/postgresql-client postgresql://user:pass@host:5432/db

docker run -it --rm jbergknoff/postgresql-client postgresql://githubstats:githubstats@localhost:5432/githubstats
```