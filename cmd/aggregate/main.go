package main

import (
	"flag"
	"log"

	"github.com/grafana/github-repo-metrics/pkg/archive"
)

var (
	database         = ""
	connectionString = ""
)

func main() {
	flag.StringVar(&database, "database", "", "database type")
	flag.StringVar(&connectionString, "connectionString", "", "connection string")
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

	aggregator := archive.NewAggregator(engine)
	err = aggregator.Aggregate()
	if err != nil {
		log.Fatalf("failed to aggregate events. error %v", err)
	}
}
