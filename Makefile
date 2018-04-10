all: deps build

dev:
	rm -rf test.db && go build -o main . && ./main

prod:
	go build -o main . && ./main

download-arch-files:
	cd archive
	wget http://data.githubarchive.org/2015-01-{01..30}-{0..23}.json.gz
