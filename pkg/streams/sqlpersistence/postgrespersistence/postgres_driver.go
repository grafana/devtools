package sqlpersistence

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"

	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/log"
	"github.com/grafana/devtools/pkg/streams/sqlpersistence"
	"github.com/lib/pq"
)

func init() {
	sqlpersistence.Register("postgres", new())
}

type postgresDriver struct {
	logger log.Logger
}

func new() *postgresDriver {
	return &postgresDriver{
		logger: log.New(),
	}
}

func (sp *postgresDriver) Init(logger log.Logger) error {
	loggerInstance := logger.New("logger", "postgres-persistence")
	sp.logger = loggerInstance
	return nil
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

func (sp *postgresDriver) PersistStream(tx *sql.Tx, t *sqlpersistence.Table, stream streams.Readable) (int64, error) {
	stmt, err := tx.Prepare(pq.CopyIn(t.TableName, t.GetColumnNames()...))
	if err != nil {
		return 0, err
	}

	rowsAffected := int64(0)
	for msg := range stream {
		_, err = stmt.Exec(t.GetColumnValues(msg)...)
		rowsAffected++
		if err != nil {
			return 0, err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return 0, err
	}

	err = stmt.Close()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
