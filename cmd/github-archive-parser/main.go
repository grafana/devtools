package main

import (
	"flag"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grafana/devtools/pkg/archive"
	_ "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var (
	simpleDateFormat = "2006-01-02"
)

func main() {
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
	)

	flag.StringVar(&database, "database", "", "database type")
	flag.StringVar(&connectionString, "connstring", "", "")
	flag.StringVar(&archiveUrl, "archiveUrl", "https://data.githubarchive.org/%d-%02d-%02d-%d.json.gz", "")
	flag.StringVar(&startDateFlag, "startDateFlag", "2015-01-01", "")
	flag.StringVar(&stopDateFlag, "stopDateFlag", "", "")
	flag.StringVar(&orgNamesFlag, "orgNames", "grafana", "comma sepearted list of orgs to download all events for")
	flag.DurationVar(&maxDuration, "maxDuration", time.Minute*10, "")
	flag.BoolVar(&overrideAllFiles, "overrideAllFiles", false, "overrides all files instead of just those missing")
	flag.Parse()

	startDate, err := time.Parse(simpleDateFormat, startDateFlag)
	if err != nil {
		log.Fatalf("could not parse start date. error: %v", err)
	}

	var stopDate time.Time
	if stopDateFlag == "" {
		stopDate = time.Now()
	} else {
		stopDate, err = time.Parse(simpleDateFormat, stopDateFlag)
		if err != nil {
			log.Fatalf("could not parse stop date. error: %v", err)
		}
	}

	orgNames = strings.Split(orgNamesFlag, ",")
	engine, err := archive.InitDatabase(database, connectionString)
	if err != nil {
		log.Fatalf("migration failed. error: %v", err)
	}

	doneChan := make(chan time.Time)
	ad := archive.NewArchiveDownloader(engine, overrideAllFiles, archiveUrl, orgNames, startDate, stopDate, doneChan)
	go func() {
		<-time.After(maxDuration)
		close(doneChan)
	}()

	err = ad.DownloadEvents()
	if err != nil {
		log.Fatalf("failed to download archive files. error: %v", err)
	}
}