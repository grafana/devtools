all: deps build

VERSION := v2

clean:
	rm -rf archive
	rm -rf bin/*
	rm -rf test.db

build:
	go build -o bin/aggregate ./cmd/aggregate/.
	go build -o bin/archive ./cmd/archive/.

archive: build
	./bin/archive -database="sqlite3" -connectionString="./test.db" -maxDurationMin=15 -archiveUrl="http://localhost:8100/%d-%02d-%02d-%d.json.gz"

docker-build:
	cp ./bin/archive ./docker/github-stats-archive
	cp ./bin/aggregate ./docker/github-stats-aggregate
	docker build --tag grafana/github-repo-metrics-archive:${VERSION} ./docker/archive
	docker build --tag grafana/github-repo-metrics-aggregate:${VERSION} ./docker/aggregate

docker-push:
	docker push grafana/github-repo-metrics:${VERSION}

archive-prod: build
	./bin/archive -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://data.githubarchive.org/%d-%02d-%02d-%d.json.gz"

aggregate: build
	./bin/aggregate -database="sqlite3" -connectionString="./test2.db"