package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/archive"
)

func main() {
	var (
		database         string
		connectionString string
		maxDuration      time.Duration
	)
	flag.StringVar(&database, "database", "", "database type to use")
	flag.StringVar(&connectionString, "connstring", "", "connectionstring to the database where the aggregator can find raw data")
	flag.DurationVar(&maxDuration, "maxDuration", time.Minute*10, "max duration for how long the process should be running")
	flag.Parse()

	engine, err := archive.InitDatabase(database, connectionString)
	if err != nil {
		log.Fatalf("migration failed. error: %v", err)
	}

	doneChan := make(chan time.Time)
	aggregator := archive.NewAggregator(engine, doneChan)
	go func() {
		<-time.After(time.Duration(maxDuration))
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
