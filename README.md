# Github repo metrics

A bundle of CLI applications for gathering/extracting data from github
and turning it into actionable data.

download arch files and serve locally
```bash
cd testdata
wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
go run main.go
```

## Test run locally

### Postgres

```bash
docker run --name gh-postgres --network host -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=github_stats -d postgres:10.5
docker-compose -f devenv/postgres-archive-parser-sample.yaml up
docker-compose -f devenv/postgres-event-aggreagor-sample.yaml up
```

### MySQL

```bash
docker run --name gh-mysql --network host -e MYSQL_ROOT_PASSWORD=rootpass -e MYSQL_USER=test -e MYSQL_PASSWORD=test -e MYSQL_DATABASE=github_stats -d mysql:5.7
docker-compose -f devenv/mysql-archive-parser-sample.yaml up
docker-compose -f devenv/mysql-event-aggreagor-sample.yaml up
```

## Github event aggregation database

**Create read only postgres user:**

```bash
$ psql -h <hostname> <database> <user>
> CREATE USER <readonly user> WITH ENCRYPTED PASSWORD '<pwd>';
> GRANT CONNECT ON DATABASE <database> TO <readonly user>;
> GRANT USAGE ON SCHEMA public TO <readonly user>;
```


