package sqlpersistence

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"

	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/sqlpersistence"
	"github.com/lib/pq"
)

func init() {
	sqlpersistence.Register("postgres", &postgresDriver{})
}

type postgresDriver struct {
}

func (sp *postgresDriver) DropTableIfExists(tx *sql.Tx, t *sqlpersistence.Table) error {
	_, err := tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, pq.QuoteIdentifier(t.TableName)))
	if err != nil {
		return err
	}

	return nil
}

func (sp *postgresDriver) CreateTableIfNotExists(tx *sql.Tx, t *sqlpersistence.Table) error {
	var createTable bytes.Buffer
	createTable.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ( ", pq.QuoteIdentifier(t.TableName)))
	primaryKeys := []string{}
	for _, c := range t.Columns {
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
	_, err := tx.Exec(createTable.String())

	if err != nil {
		return err
	}

	return nil
}

func (sp *postgresDriver) PersistStream(tx *sql.Tx, t *sqlpersistence.Table, stream streams.Readable) error {
	stmt, err := tx.Prepare(pq.CopyIn(t.TableName, t.GetColumnNames()...))
	if err != nil {
		return err
	}

	for msg := range stream {
		_, err = stmt.Exec(t.GetColumnValues(msg)...)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	return nil
}
