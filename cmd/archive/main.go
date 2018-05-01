package main

import (
	"flag"
	"log"

	"github.com/grafana/github-repo-metrics/pkg/archive"
	_ "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

var (
	connectionString = ""
	database         = ""
	archiveUrl       = ""
	startDate        = "2014-01-01"

	repoIds = []int64{15111821}
)

func main() {
	flag.StringVar(&connectionString, "connectionString", "", "description")
	flag.StringVar(&database, "database", "", "description")
	flag.StringVar(&archiveUrl, "archiveUrl", "default?", "description")
	flag.StringVar(&startDate, "startDate", "", "start date for parsing events")
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

	ad := archive.NewArchiveDownloader(engine, archiveUrl, repoIds)
	ad.DownloadEvents()
}
