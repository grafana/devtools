package main

import (
	"flag"
	"log"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/archive"
	_ "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

var (
	connectionString = ""
	database         = ""
	archiveUrl       = ""
	startDateFlag    = ""
	stopDateFlag     = ""
	maxDurationMin   int64

	repoIds = []int64{15111821}

	simpleDateFormat = "2006-01-02"
)

func main() {
	flag.StringVar(&connectionString, "connectionString", "", "description")
	flag.StringVar(&database, "database", "", "description")
	flag.StringVar(&archiveUrl, "archiveUrl", "default?", "description")
	flag.StringVar(&startDateFlag, "startDate", "2015-01-01", "start date for parsing events")
	flag.StringVar(&stopDateFlag, "stopDate", "2015-01-03", "last date the program should download events for")
	flag.Int64Var(&maxDurationMin, "maxDurationMin", 60, "maxium time this application should run. will shutdown gracefully after")
	flag.Parse()

	// f, err := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.Fatalf("error opening file: %v", err)
	// }
	// defer f.Close()

	// log.SetOutput(f)

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
		close(doneChan)
	}()

	err = ad.DownloadEvents()
	if err != nil {
		log.Fatalf("failed to download archive files. error: %v", err)
	}
}
