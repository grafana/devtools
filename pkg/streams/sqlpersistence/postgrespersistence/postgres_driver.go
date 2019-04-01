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
	dropTableSQL := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, pq.QuoteIdentifier(t.TableName))
	_, err := tx.Exec(dropTableSQL)
	if err != nil {
		sp.logger.Debug("failed to drop database table", "table", t.TableName, "sql", dropTableSQL)
		return err
	}

	return nil
}

func (sp *postgresDriver) CreateTableIfNotExists(tx *sql.Tx, t *sqlpersistence.Table) error {
	var createTableSQL bytes.Buffer
	createTableSQL.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ( ", pq.QuoteIdentifier(t.TableName)))
	primaryKeys := []string{}
	for _, c := range t.Columns {
		createTableSQL.WriteString(pq.QuoteIdentifier(c.Name))
		createTableSQL.WriteString(" ")

		columnType, err := getColumnType(c)
		if err != nil {
			return err
		}

		createTableSQL.WriteString(columnType)

		if !c.IsNullable {
			createTableSQL.WriteString(" NOT NULL")
		}

		createTableSQL.WriteString(", ")

		if c.IsPrimaryKey {
			primaryKeys = append(primaryKeys, pq.QuoteIdentifier(c.Name))
		}
	}

	createTableSQL.WriteString("PRIMARY KEY(")
	createTableSQL.WriteString(strings.Join(primaryKeys, ","))
	createTableSQL.WriteString("))")
	_, err := tx.Exec(createTableSQL.String())

	if err != nil {
		sp.logger.Debug("failed to create database table", "table", t.TableName, "sql", createTableSQL.String())
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
		values := t.GetColumnValues(msg)
		if len(values) == 0 {
			continue
		}

		_, err = stmt.Exec(values...)
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

func getColumnType(c *sqlpersistence.Column) (string, error) {
	switch c.Type {
	case sqlpersistence.ColumnTypeInteger:
		switch c.Length {
		case 32:
			return "INTEGER", nil
		case 64:
			return "BIGINT", nil
		}
	case sqlpersistence.ColumnTypeFloat:
		return "REAL", nil
	case sqlpersistence.ColumnTypeString:
		columnType := "TEXT"

		// if c.IsUnicode {
		// 	columnType += " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
		// }

		return columnType, nil
	case sqlpersistence.ColumnTypeBoolean:
		return "BOOLEAN", nil
	}

	return "", fmt.Errorf("column type %s not supported", c.Type)
}
