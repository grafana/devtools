package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"

	//_ "github.com/go-xorm/core"
	"github.com/go-xorm/xorm"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var archiveUrlPattern = `http://localhost:8100/%s-%s-%s-%d.json.gz`

var eventCount = 0
var allEvents []GithubEventJson
var x *xorm.Engine

func downloadEvents() {

	//x, err := xorm.NewEngine("postgres", "user=githubstats password=githubstats host=localhost port=5432 dbname=githubstats sslmode=disable")
	x, err := xorm.NewEngine("sqlite3", "./test.db")
	logOnError(err, "create db connection")

	mig := migrator.NewMigrator(x)

	migrationLogV1 := migrator.Table{
		Name: "migration_log",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "migration_id", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "sql", Type: migrator.DB_Text},
			{Name: "success", Type: migrator.DB_Bool},
			{Name: "error", Type: migrator.DB_Text},
			{Name: "timestamp", Type: migrator.DB_DateTime},
		},
	}

	mig.AddMigration("create migration_log table", migrator.NewAddTableMigration(migrationLogV1))

	archiveFile := migrator.Table{
		Name: "archive_file",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "date", Type: migrator.DB_Text},
			{Name: "filename", Type: migrator.DB_Text},
		},
	}

	mig.AddMigration("create archive file table", migrator.NewAddTableMigration(archiveFile))

	githubEvent := migrator.Table{
		Name: "github_event",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true},
			{Name: "type", Type: migrator.DB_Text},
			{Name: "repo_id", Type: migrator.DB_BigInt},
			{Name: "date", Type: migrator.DB_Text},
		},
	}

	mig.AddMigration("create github event table", migrator.NewAddTableMigration(githubEvent))

	err = mig.Start()
	logOnError(err, "migration")

	var urls []string

	years := []string{"2018"}
	months := []string{"01"}
	//days := []string{"01", "02", "03", "04", "05"}
	days := []string{"01"}
	for _, y := range years {
		for _, m := range months {
			for _, d := range days {
				for hour := 1; hour < 24; hour++ {
					urls = append(urls, fmt.Sprintf(archiveUrlPattern, y, m, d, hour))
				}
			}
		}
	}

	start := time.Now()
	for _, u := range urls {
		download(u)
	}

	fmt.Println("filtred event: ", len(allEvents))
	fmt.Println("event count: ", eventCount)
	fmt.Println("ellapsed: ", time.Since(start))
}

func download(url string) {
	fmt.Printf("downloading: %s\n", url)
	res, err := http.Get(url)
	logOnError(err, "failed to download json file")

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)

	zr, err := gzip.NewReader(buf)
	logOnError(err, "parsing compress content")

	var events []GithubEventJson

	for {
		zr.Multistream(false)

		scanner := bufio.NewScanner(zr)
		scanner.Buffer([]byte(""), 1024*1024) //increase buffer limit

		for scanner.Scan() {
			ge := GithubEventJson{}
			err := json.Unmarshal([]byte(scanner.Text()), &ge)
			logOnError(err, "parsing json")

			eventCount++
			if ge.Type == "PushEvent" {
				//allEvents = append(allEvents, ge)
				events = append(events, ge)
			}
		}

		err := scanner.Err()
		if err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			break
		}

		if scanner.Err() == io.EOF {
			fmt.Println("scanner EOF")
			break
		}

		err = zr.Reset(buf)
		if err == io.EOF {
			break
		}
	}

	for _, e := range events {
		dbModel := e.CreateGithubEvent()
		fmt.Printf("should insert %+v\n", dbModel)
		//aff, err := x.Insert(dbModel)
		//sess := x.NewSession()
		//_, err = x.Exec("INSERT INTO github_event (id, type, repo_id, date) values (?,?,?,?)", dbModel.Id, dbModel.Type, dbModel.RepoId, dbModel.Date)
		//_, err = sess.Exec("INSERT INTO github_event (type) values (?)", dbModel.Type)
		//logOnError(err, "insert into db")

		engine, err := xorm.NewEngine("sqlite3", "./test.db")
		_, err = engine.Insert(dbModel)

		//err = sess.Commit()
		logOnError(err, "insert")
	}

	//inserted, err := x.Insert(events)

	//logOnError(err, "insert into db")

	allEvents = append(allEvents, events...)
	logOnError(zr.Close(), "closing buffer")
}
