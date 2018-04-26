all: deps build

archive:
	go build -o main ./cmd/archive/.
	./main -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://localhost:8100/%d-%02d-%02d-%d.json.gz"

archive-prod:
	go build -o main ./cmd/archive/.
	./main -database="sqlite3" -connectionString="./test.db" -archiveUrl="http://data.githubarchive.org/%d-%02d-%02d-%d.json.gz"

aggregate:
	go build -o main ./cmd/aggregate/.
	./main -database="sqlite3" -connectionString="./test.db" 