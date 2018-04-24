all: deps build

arch:
	go build -o main ./pkg/arch/.
	./main -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://localhost:8100/%d-%02d-%02d-%d.json.gz"

archprod:
	go build -o main ./pkg/arch/.
	./main -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://data.githubarchive.org/%d-%02d-%02d-%d.json.gz"

download-arch-files:
	cd archive
	wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
