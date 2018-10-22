package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/archive"
	_ "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

var (
	repoIds = []int64{15111821}

	simpleDateFormat = "2006-01-02"
)

func main() {
	database := lookupEnvString("STATS_DB")
	connectionString := lookupEnvString("STATS_CONNSTRING")
	archiveUrl := lookupEnvString("STATS_ARCHIVEURL")
	startDateFlag := lookupEnvString("STATS_STARTDATEFLAG")
	stopDateFlag := lookupEnvString("STATS_STOPDATEFLAG")
	maxDurationMinString := lookupEnvString("STATS_MAXDURATION")
	maxDurationMin, err := strconv.ParseInt(maxDurationMinString, 10, 64)
	if err != nil {
		log.Fatalf("could not parse %s to int64", maxDurationMinString)
	}

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
		<-time.After(time.Duration(maxDurationMin * int64(time.Minute)))
		//<-time.After(time.Duration(time.Second * 3))
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
