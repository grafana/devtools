package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/archive"
	_ "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

var (
	repoIds = []int64{15111821}

	simpleDateFormat = "2006-01-02"
)

func main() {
	var (
		database         string
		connectionString string
		archiveUrl       string
		startDateFlag    string
		stopDateFlag     string
		maxDuration      time.Duration
	)

	flag.StringVar(&database, "database", "", "database type")
	flag.StringVar(&connectionString, "connstring", "", "")
	flag.StringVar(&archiveUrl, "archiveUrl", "https://data.githubarchive.org/%d-%02d-%02d-%d.json.gz", "")
	flag.StringVar(&startDateFlag, "startDateFlag", "2015-01-01", "")
	flag.StringVar(&stopDateFlag, "stopDateFlag", "", "")
	flag.DurationVar(&maxDuration, "maxDuration", time.Minute*10, "")
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

	engine, err := archive.InitDatabase(database, connectionString)
	if err != nil {
		log.Fatalf("migration failed. error: %v", err)
	}

	doneChan := make(chan time.Time)
	ad := archive.NewArchiveDownloader(engine, archiveUrl, repoIds, startDate, stopDate, doneChan)
	go func() {
		<-time.After(maxDuration)
		close(doneChan)
	}()

	err = ad.DownloadEvents()
	if err != nil {
		log.Fatalf("failed to download archive files. error: %v", err)
	}
}

func lookupEnvString(name string) string {
	value, exists := os.LookupEnv(name)
	if !exists {
		log.Fatalf("missing env variable %s", name)
	}

	return value
}
