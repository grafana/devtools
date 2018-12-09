package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	ghevents "github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/githubstats"
	"github.com/grafana/devtools/pkg/streams/pkg/streamprojections"
	"github.com/grafana/devtools/pkg/streams/pkg/streams"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "user=test password=test host=localhost port=5432 dbname=github_stats sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	defer db.Close()

	s := streams.New()
	projectionEngine := streamprojections.New(s, NewPostgresStreamPersister(db))
	githubstats.RegisterProjections(projectionEngine)

	events, errors := readEvents("output")
	go printErrorSummary(errors)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-s.Start()
		wg.Done()
	}()

	s.Publish(githubstats.GithubEventStream, events)
	wg.Wait()
}

var parseFileDate = regexp.MustCompile(`^(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})-(?P<hour>\d{1,2}).json$`)

func readEvents(dir string) (streams.Readable, <-chan error) {
	out := make(chan streams.T)
	outErr := make(chan error)

	var wg sync.WaitGroup
	wg.Add(1)

	go func(dir string) {
		files, err := filepath.Glob(path.Join(dir, "*.json"))
		if err != nil {
			outErr <- err
			return
		}

		fileTimestampMap := map[float64]string{}

		for _, fp := range files {
			_, f := path.Split(fp)
			n1 := parseFileDate.SubexpNames()
			r2 := parseFileDate.FindAllStringSubmatch(f, -1)[0]
			md := map[string]string{}
			for i, n := range r2 {
				md[n1[i]] = n
			}

			t, err := time.Parse("2006-01-02T15", fmt.Sprintf("%s-%s-%sT%02s", md["year"], md["month"], md["day"], md["hour"]))
			if err != nil {
				outErr <- err
				continue
			}

			fileTimestampMap[float64(t.Unix())] = fp
		}

		sortedTimestamps := []float64{}
		for k := range fileTimestampMap {
			sortedTimestamps = append(sortedTimestamps, k)
		}
		sort.Float64s(sortedTimestamps)

		for _, ts := range sortedTimestamps {
			fp := fileTimestampMap[ts]
			file, err := os.Open(fp)
			if err != nil {
				outErr <- err
			}

			bufferedEventMap := map[string]*ghevents.Event{}

			reader := bufio.NewReader(file)
			d := json.NewDecoder(reader)
			for {
				var evt ghevents.Event
				err := d.Decode(&evt)
				if err == io.EOF {
					file.Close()
					break
				} else if err != nil {
					file.Close()
					outErr <- err
					break
				}

				bufferedEventMap[evt.ID] = &evt
			}
			file.Close()

			sortedEvents := []string{}
			for k := range bufferedEventMap {
				sortedEvents = append(sortedEvents, k)
			}
			sort.Strings(sortedEvents)

			for _, id := range sortedEvents {
				out <- bufferedEventMap[id]
			}
		}

		wg.Done()
	}(dir)

	go func() {
		wg.Wait()
		close(out)
		close(outErr)
	}()

	return out, outErr
}

func printErrorSummary(errors <-chan error) {
	for err := range errors {
		fmt.Println("Receive error", "err", err)
	}
}

type postgresStreamPersister struct {
	*streamprojections.StreamPersisterBase
	db *sql.DB
}

func NewPostgresStreamPersister(db *sql.DB) streamprojections.StreamPersister {
	return &postgresStreamPersister{
		StreamPersisterBase: streamprojections.NewStreamPersisterBase(),
		db:                  db,
	}
}

func (sp *postgresStreamPersister) Register(name string, objTemplate interface{}) {
	sp.StreamPersisterBase.Register(name, objTemplate)
	tx, err := sp.db.Begin()
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, pq.QuoteIdentifier(name)))
	if err != nil {
		fmt.Println(err)
		err = tx.Rollback()
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	var createTable bytes.Buffer
	createTable.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ( ", pq.QuoteIdentifier(name)))
	primaryKeys := []string{}
	for _, c := range sp.StreamsToPersist[name].Columns {
		createTable.WriteString(pq.QuoteIdentifier(c.ColumnName) + " ")
		createTable.WriteString(c.ColumnType + " ")
		createTable.WriteString("NOT NULL, ")
		if c.PrimaryKey {
			primaryKeys = append(primaryKeys, pq.QuoteIdentifier(c.ColumnName))
		}
	}
	createTable.WriteString("PRIMARY KEY(")
	createTable.WriteString(strings.Join(primaryKeys, ","))
	createTable.WriteString("))")
	_, err = tx.Exec(createTable.String())

	if err != nil {
		fmt.Println(err)
		err = tx.Rollback()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (sp *postgresStreamPersister) Persist(name string, stream streams.Readable) {
	s := sp.StreamsToPersist[name]
	txn, err := sp.db.Begin()
	if err != nil {
		fmt.Println(err)
		return
	}

	stmt, err := txn.Prepare(pq.CopyIn(name, s.GetColumnNames()...))
	if err != nil {
		fmt.Println(err)
		return
	}

	for msg := range stream {
		_, err = stmt.Exec(s.GetColumnValues(msg)...)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = stmt.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = txn.Commit()
	if err != nil {
		fmt.Println(err)
	}
}
