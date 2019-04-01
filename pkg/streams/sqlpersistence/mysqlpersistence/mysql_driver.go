package mysqlpersistence

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	// make sure to load mysql driver

	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/log"
	"github.com/grafana/devtools/pkg/streams/sqlpersistence"
)

func init() {
	sqlpersistence.Register("mysql", new())
}

type mySqlLogger struct {
	logger log.Logger
}

func (l *mySqlLogger) Print(v ...interface{}) {
	l.logger.Error("critical error", v...)
}

type mySqlDriver struct {
	logger log.Logger
}

func new() *mySqlDriver {
	return &mySqlDriver{
		logger: log.New(),
	}
}

func (sp *mySqlDriver) Init(logger log.Logger) error {
	loggerInstance := logger.New("logger", "mysql-persistence")
	sp.logger = loggerInstance
	mysql.SetLogger(&mySqlLogger{logger: loggerInstance})
	return nil
}

func (sp *mySqlDriver) DropTableIfExists(tx *sql.Tx, t *sqlpersistence.Table) error {
	dropTableSQL := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, t.TableName)
	_, err := tx.Exec(dropTableSQL)
	if err != nil {
		sp.logger.Debug("failed to drop database table", "table", t.TableName, "sql", dropTableSQL)
		return err
	}

	return nil
}

func (sp *mySqlDriver) CreateTableIfNotExists(tx *sql.Tx, t *sqlpersistence.Table) error {
	var createTableSQL bytes.Buffer
	createTableSQL.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", t.TableName))
	primaryKeys := []string{}
	for _, c := range t.Columns {
		createTableSQL.WriteString(c.Name)
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
			primaryKeys = append(primaryKeys, c.Name)
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

func (sp *mySqlDriver) PersistStream(tx *sql.Tx, t *sqlpersistence.Table, stream streams.Readable) (int64, error) {
	buf := &bytes.Buffer{}
	buf.WriteString("INSERT INTO ")
	buf.WriteString(t.TableName)
	buf.WriteString(" (")
	buf.WriteString(strings.Join(t.GetColumnNames(), ","))
	buf.WriteString(") VALUES ")
	initialSQL := buf.String()

	values := []interface{}{}

	preparedArgs := []string{}
	for n := 0; n < len(t.GetColumnNames()); n++ {
		preparedArgs = append(preparedArgs, "?")
	}
	preparedSQLStr := "(" + strings.Join(preparedArgs, ",") + ")"
	sql := ""
	processedRows := int64(0)
	rowsAffected := int64(0)

	for msg := range stream {
		colValues := t.GetColumnValues(msg)
		if len(colValues) == 0 {
			continue
		}
		values = append(values, colValues...)

		if processedRows != 0 {
			sql += ","
		}

		sql += preparedSQLStr
		processedRows++
		rowsAffected++

		if processedRows > 999 {
			stmt, err := tx.Prepare(initialSQL + sql)
			if err != nil {
				return 0, err
			}

			_, err = stmt.Exec(values...)
			if err != nil {
				return 0, err
			}

			err = stmt.Close()
			if err != nil {
				return 0, err
			}

			processedRows = 0
			sql = ""
			values = []interface{}{}
		}
	}

	if len(values) == 0 {
		return rowsAffected, nil
	}

	stmt, err := tx.Prepare(initialSQL + sql)
	if err != nil {
		return 0, err
	}

	_, err = stmt.Exec(values...)
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
		columnType := fmt.Sprintf("VARCHAR(%d)", c.Length)

		if c.IsUnicode {
			columnType += " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
		}

		return columnType, nil
	case sqlpersistence.ColumnTypeBoolean:
		return "BOOLEAN", nil
	}

	return "", fmt.Errorf("column type %s not supported", c.Type)
}
