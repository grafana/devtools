all: deps build

test:
	rm -rf test.db && go build -o main . && ./main -database="sqlite3" -connectionString="./test.db"

prod:
	go build -o main . && ./main

download-arch-files:
	cd archive
	wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
