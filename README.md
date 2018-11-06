# Github repo metrics

A bundle of CLI applications for gathering/extracting data from github 
and turning it into actionable data.

download arch files and serve locally
```bash
cd testdata
wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
go run main.go
```

Run binaries locally
```bash

docker-compose -f devenv/archive.yaml up
docker-compose -f devenv/aggreagor.yaml up

```
