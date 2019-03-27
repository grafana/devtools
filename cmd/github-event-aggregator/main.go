package main

import (
	"flag"
	"log"
	"sync"
	"time"

	"github.com/grafana/devtools/pkg/streams/sqlpersistence"

	"github.com/grafana/devtools/pkg/archive"
	"github.com/grafana/devtools/pkg/githubstats"
	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/projections"
	_ "github.com/grafana/devtools/pkg/streams/sqlpersistence/mysqlpersistence"
	_ "github.com/grafana/devtools/pkg/streams/sqlpersistence/postgrespersistence"
)

func main() {
	start := time.Now()

	var (
		database             string
		fromConnectionString string
		toConnectionString   string
		limit                int64
		readonlyUser         string
	)
	flag.StringVar(&database, "database", "", "database type")
	flag.StringVar(&fromConnectionString, "fromConnectionstring", "", "")
	flag.StringVar(&toConnectionString, "toConnectionstring", "", "")
	flag.Int64Var(&limit, "limit", 5000, "")
	flag.StringVar(&readonlyUser, "readonlyUser", "gh_reader", "The readonly database user to grant select permission on generated tables")
	flag.Parse()

	streamPersister, err := sqlpersistence.Open(database, toConnectionString)
	if err != nil {
		log.Fatalln("Failed to open sql stream persister", err)
	}

	defer func() {
		err := streamPersister.Close()
		if err != nil {
			log.Fatalf("failed to close stream persister. error: %v", err)
		}
	}()

	s := streams.New()
	projectionEngine := projections.New(s, streamPersister)
	githubstats.RegisterProjections(projectionEngine)

	engine, err := archive.InitDatabase(database, fromConnectionString)
	if err != nil {
		log.Fatalf("migration failed. error: %v", err)
	}

	reader := archive.NewArchiveReader(engine, limit)

	events, errors := reader.ReadAllEvents()
	go printErrorSummary(errors)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-s.Start()
		wg.Done()
	}()

	s.Publish(githubstats.GithubEventStream, events)
	wg.Wait()

	elapsed := time.Since(start)
	log.Printf("Done. Took %s\n", elapsed)
}

func printErrorSummary(errors <-chan error) {
	for err := range errors {
		log.Println("Receive error", "err", err)
	}
}
