package main

import (
	"flag"
	"os"
	"runtime"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grafana/devtools/pkg/archive"
	"github.com/grafana/devtools/pkg/log15adapter"
	"github.com/grafana/devtools/pkg/streams/log"
	_ "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	"github.com/inconshreveable/log15"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var (
	simpleDateFormat = "2006-01-02"
)

func main() {
	start := time.Now()

	var (
		database         string
		connectionString string
		archiveUrl       string
		startDateFlag    string
		stopDateFlag     string
		orgNamesFlag     string
		orgNames         []string
		maxDuration      time.Duration
		overrideAllFiles bool
		numWorkers       int
		verboseLogging   bool
	)

	flag.StringVar(&database, "database", "", "database type")
	flag.StringVar(&connectionString, "connstring", "", "")
	flag.StringVar(&archiveUrl, "archiveUrl", "https://data.githubarchive.org/%d-%02d-%02d-%d.json.gz", "")
	flag.StringVar(&startDateFlag, "startDateFlag", "2015-01-01", "")
	flag.StringVar(&stopDateFlag, "stopDateFlag", "", "")
	flag.StringVar(&orgNamesFlag, "orgNames", "grafana", "comma sepearted list of orgs to download all events for")
	flag.DurationVar(&maxDuration, "maxDuration", time.Minute*10, "")
	flag.BoolVar(&overrideAllFiles, "overrideAllFiles", false, "overrides all files instead of just those missing")
	flag.IntVar(&numWorkers, "number of workers to spawn", runtime.NumCPU(), "defaults to the number of CPUs")
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

	startDate, err := time.Parse(simpleDateFormat, startDateFlag)
	if err != nil {
		logger.Fatal("could not parse start date", "error", err)
	}

	var stopDate time.Time
	if stopDateFlag == "" {
		stopDate = time.Now()
	} else {
		stopDate, err = time.Parse(simpleDateFormat, stopDateFlag)
		if err != nil {
			logger.Fatal("could not parse stop date", "error", err)
		}
	}

	orgNames = strings.Split(orgNamesFlag, ",")
	engine, err := archive.InitDatabase(database, connectionString)
	if err != nil {
		logger.Fatal("migration failed", "error", err)
	}

	doneChan := make(chan time.Time)
	ad := archive.NewArchiveDownloader(engine, overrideAllFiles, archiveUrl, orgNames, startDate, stopDate, numWorkers, doneChan)
	ad.SetLogger(logger)

	go func() {
		<-time.After(maxDuration)
		close(doneChan)
	}()

	err = ad.DownloadEvents()
	if err != nil {
		logger.Fatal("failed to download archive files", "error", err)
	}

	logger.Info("done", "took", time.Since(start))
}
