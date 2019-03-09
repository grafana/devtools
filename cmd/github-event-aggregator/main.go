package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/grafana/devtools/pkg/archive"
	"github.com/grafana/devtools/pkg/githubstats"
	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/projections"
	"github.com/lib/pq"
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

	toDb, err := sql.Open(database, toConnectionString)
	if err != nil {
		log.Fatalln("Failed to connect to event aggregator database", err)
	}
	defer toDb.Close()

	if err = toDb.Ping(); err != nil {
		log.Fatalln("Error: Could not establish a connection with the event aggregator database", err)
	}

	s := streams.New()
	projectionEngine := projections.New(s, NewPostgresStreamPersister(toDb, readonlyUser))
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

type postgresStreamPersister struct {
	*projections.StreamPersisterBase
	db           *sql.DB
	readonlyUser string
}

func NewPostgresStreamPersister(db *sql.DB, readonlyUser string) projections.StreamPersister {
	return &postgresStreamPersister{
		StreamPersisterBase: projections.NewStreamPersisterBase(),
		db:                  db,
		readonlyUser:        readonlyUser,
	}
}

func (sp *postgresStreamPersister) Register(name string, objTemplate interface{}) {
	sp.StreamPersisterBase.Register(name, objTemplate)
	tx, err := sp.db.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	_, err = tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, pq.QuoteIdentifier(name)))
	if err != nil {
		log.Println(err)
		err = tx.Rollback()
		if err != nil {
			log.Println(err)
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
		log.Println(err)
		err = tx.Rollback()
		if err != nil {
			log.Println(err)
			return
		}
	}

	err = sp.grantSelectPermissionOnTable(tx, name)
	if err != nil {
		log.Println(err)
		err = tx.Rollback()
		if err != nil {
			log.Println(err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println(err)
		return
	}
}

func (sp *postgresStreamPersister) grantSelectPermissionOnTable(tx *sql.Tx, table string) error {
	_, err := tx.Exec(fmt.Sprintf(`GRANT SELECT ON %s TO %s`, pq.QuoteIdentifier(table), sp.readonlyUser))
	if err != nil {
		return err
	}

	return nil
}

func (sp *postgresStreamPersister) Persist(name string, stream streams.Readable) {
	s := sp.StreamsToPersist[name]
	txn, err := sp.db.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	stmt, err := txn.Prepare(pq.CopyIn(name, s.GetColumnNames()...))
	if err != nil {
		log.Println(err)
		return
	}

	for msg := range stream {
		_, err = stmt.Exec(s.GetColumnValues(msg)...)
		if err != nil {
			log.Println(err)
			continue
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		log.Println(err)
		return
	}

	err = stmt.Close()
	if err != nil {
		log.Println(err)
		return
	}

	err = txn.Commit()
	if err != nil {
		log.Println(err)
	}
}
