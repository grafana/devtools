# Local GitHub Archive Server

Server behaving like data.gharchive.org, but serves files from the filesystem.
Can be handy to use when you have a set of pre-filtered GitHub Archive json
files and quickly want to populate the database with events compared to going
against gharchive.org which takes quite a while to process.

Content are served on port 8000 by default, but can be changed using the `port` argument.

```bash
./local-gharchive-server -path=/some/path/where/gharchive/json/files/are/stored

./github-archive-parser -database=mysql -connstring=test:test@tcp(localhost:3306)/github_stats -maxDuration=10h -orgNames=grafana -overrideAllFiles=false -archiveUrl=http://localhost:8000/%d-%02d-%02d-%d.json.gz

```