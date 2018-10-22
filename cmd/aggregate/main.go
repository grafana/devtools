package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/archive"
)

func main() {
	database := lookupEnvString("STATS_DB")
	connectionString := lookupEnvString("STATS_CONNSTRING")
	maxDurationMinString := lookupEnvString("STATS_MAXDURATION")
	maxDurationMin, err := strconv.ParseInt(maxDurationMinString, 10, 64)
	if err != nil {
		log.Fatalf("could not parse %s to int64", maxDurationMinString)
	}

	engine, err := archive.InitDatabase(database, connectionString)
	if err != nil {
		log.Fatalf("migration failed. error: %v", err)
	}

	doneChan := make(chan time.Time)
	aggregator := archive.NewAggregator(engine, doneChan)
	go func() {
		<-time.After(time.Duration(maxDurationMin * int64(time.Minute)))
		close(doneChan)
	}()

	err = aggregator.Aggregate()
	if err != nil {
		log.Fatalf("failed to aggregate events. error %v", err)
	}
}

func lookupEnvString(name string) string {
	value, exists := os.LookupEnv(name)
	if !exists {
		log.Fatalf("missing env variable %s", name)
	}

	return value
}
