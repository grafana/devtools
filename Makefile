all: deps build

VERSION := v2

build:
	go build -o aggregate ./cmd/aggregate/.
	go build -o archive ./cmd/archive/.

archive: build
	./cmd/archive/archive -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://localhost:8100/%d-%02d-%02d-%d.json.gz"

docker-build:
	docker build --tag grafana/github-repo-metrics:${VERSION} .

docker-push: 
	docker push grafana/github-repo-metrics:${VERSION}

archive-prod: build
	./cmd/archive/archive -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://data.githubarchive.org/%d-%02d-%02d-%d.json.gz"

aggregate: build 
	./cmd/aggregate/aggregate -database="sqlite3" -connectionString="./test.db" 