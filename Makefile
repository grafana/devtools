all: deps build

test:
	rm -rf test.db
	go build -o main ./pkg/.
	./main -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://localhost:8100/%d-%02d-%02d-%d.json.gz"

prod:
	go build -o main .
	./main -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://data.githubarchive.org/%d-%02d-%02d-%d.json.gz"

download-arch-files:
	cd archive
	wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
