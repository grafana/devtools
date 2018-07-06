package main

import (
	"flag"
	"log"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/archive"
)

var (
	database         = ""
	connectionString = ""
	maxDurationMin   int64
)

func main() {
	flag.StringVar(&database, "database", "", "database type")
	flag.StringVar(&connectionString, "connectionString", "", "connection string")
	flag.Int64Var(&maxDurationMin, "maxDurationMin", 60, "maxium time this application should run. will shutdown gracefully after")

	flag.Parse()

	// f, err := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.Fatalf("error opening file: %v", err)
	// }
	// defer f.Close()

	// log.SetOutput(f)

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
