# Goals with this repo?

 - [ ] Download all events for grafana/grafana
 - [ ] Aggregate data per day. Issues, PR's
 - [ ] Build model of current state
 - [ ] Find new contributors.
 - [ ] follow up flagged contributors
 - [ ] see if flagged contributors keep providing.


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

Performance.
~22s per day.

22 * 365 = 8030s per year
48180s for 6 years
803min for 6 years
13h for 6 years
