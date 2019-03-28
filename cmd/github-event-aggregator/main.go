package main

import (
	"flag"
	"os"
	"sync"
	"time"

	"github.com/inconshreveable/log15"

	"github.com/grafana/devtools/pkg/log15adapter"
	"github.com/grafana/devtools/pkg/streams/sqlpersistence"

	"github.com/grafana/devtools/pkg/archive"
	"github.com/grafana/devtools/pkg/githubstats"
	"github.com/grafana/devtools/pkg/streams/log"
	"github.com/grafana/devtools/pkg/streams/memorybus"
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
		verboseLogging       bool
	)
	flag.StringVar(&database, "database", "", "database type")
	flag.StringVar(&fromConnectionString, "fromConnectionstring", "", "")
	flag.StringVar(&toConnectionString, "toConnectionstring", "", "")
	flag.Int64Var(&limit, "limit", 5000, "")
	flag.StringVar(&readonlyUser, "readonlyUser", "gh_reader", "The readonly database user to grant select permission on generated tables")
	flag.BoolVar(&verboseLogging, "verbose", false, "enable verbose logging")
	flag.Parse()

	logger := log.New()

	logLevel := log15.LvlInfo
	if verboseLogging {
		logLevel = log15.LvlDebug
	}

	log15Logger := log15.New()
	log15Logger.SetHandler(log15.LvlFilterHandler(
		logLevel, log15.StreamHandler(os.Stdout, log15adapter.GetConsoleFormat())))
	logger.AddHandler(log15adapter.New(log15Logger))

	streamPersister, err := sqlpersistence.Open(logger, database, toConnectionString)
	if err != nil {
		logger.Fatal("Failed to open sql stream persister", "error", err)
	}

	defer func() {
		err := streamPersister.Close()
		if err != nil {
			logger.Fatal("failed to close stream persister", "error", err)
		}
	}()

	bus := memorybus.New()
	bus.SetLogger(logger)

	projectionEngine := projections.New(bus, streamPersister)
	projectionEngine.SetLogger(logger)

	githubstats.RegisterProjections(projectionEngine)

	engine, err := archive.InitDatabase(database, fromConnectionString)
	if err != nil {
		logger.Fatal("migration failed", "error", err)
	}

	reader := archive.NewArchiveReader(logger, engine, limit)
	events, errors := reader.ReadAllEvents()

	go printErrorSummary(logger, errors)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-bus.Start()
		wg.Done()
	}()

	bus.Publish(githubstats.GithubEventStream, events)

	wg.Wait()

	elapsed := time.Since(start)
	logger.Info("done", "took", elapsed)
}

func printErrorSummary(logger log.Logger, errors <-chan error) {
	for err := range errors {
		logger.Error("Receive error", "error", err)
	}
}
