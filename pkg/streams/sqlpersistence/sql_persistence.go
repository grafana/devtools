package sqlpersistence

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grafana/devtools/pkg/streams"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

type Driver interface {
	DropTableIfExists(tx *sql.Tx, persistedStream *Table) error
	CreateTableIfNotExists(tx *sql.Tx, persistedStream *Table) error
	PersistStream(tx *sql.Tx, persistedStream *Table, stream streams.Readable) error
}

// Register makes a sql stream persister driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("sqlpersistence: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("sqlpersistence: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func unregisterAllDrivers() {
	driversMu.Lock()
	defer driversMu.Unlock()
	drivers = make(map[string]Driver)
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

type SQLStreamPersister struct {
	streams.StreamPersister
	Driver             Driver
	db                 *sql.DB
	registeredTablesMu sync.RWMutex
	registeredTables   map[string]*Table
}

func Open(driverName, connectionString string) (*SQLStreamPersister, error) {
	driversMu.RLock()
	driveri, ok := drivers[driverName]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("sqlpersistence: unknown driver %q (forgotten import?)", driverName)
	}

	db, err := sql.Open(driverName, connectionString)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &SQLStreamPersister{
		Driver:           driveri,
		db:               db,
		registeredTables: map[string]*Table{},
	}, nil
}

func (sp *SQLStreamPersister) Close() error {
	if sp.db != nil {
		return sp.db.Close()
	}

	return nil
}

func (sp *SQLStreamPersister) Register(name string, objTemplate interface{}) error {
	table := newTable(name)
	t := reflect.TypeOf(objTemplate).Elem()
	t.NumField()
	setColumnDataType := func(t reflect.Type, c *Column) {
		switch t.Kind() {
		case reflect.TypeOf(time.Time{}).Kind():
			c.ColumnType = "integer"
			c.ConvertFn = func(v interface{}) interface{} {
				return v.(time.Time).Unix()
			}
		case reflect.String:
			c.ColumnType = "varchar(256)"
		case reflect.Float64:
			c.ColumnType = "real"
		case reflect.Float32:
			c.ColumnType = "real"
		case reflect.Int, reflect.Int32:
			c.ColumnType = "integer"
		case reflect.Int64:
			c.ColumnType = "bigint"
		case reflect.Bool:
			c.ColumnType = "boolean"
		}
	}
	for n := 0; n < t.NumField(); n++ {
		f := t.Field(n)
		if !reflect.ValueOf(objTemplate).Elem().Field(n).CanSet() {
			continue
		}
		columnName := strings.ToLower(f.Name)
		primaryKey := false
		tag := f.Tag.Get("persist")
		if tag != "" {
			parts := strings.Split(tag, ",")
			if len(parts) > 0 && parts[0] == "-" {
				continue
			}
			if len(parts) > 0 && parts[0] != "" {
				columnName = strings.ToLower(parts[0])
			}
			for _, p := range parts {
				if p == "primarykey" {
					primaryKey = true
				}
			}
		}

		c := newColumn(f.Name, columnName, primaryKey)
		setColumnDataType(f.Type, c)
		table.Columns = append(table.Columns, c)
	}

	sp.registeredTablesMu.Lock()
	sp.registeredTables[name] = table
	sp.registeredTablesMu.Unlock()

	return sp.inTransaction(func(tx *sql.Tx) error {
		err := sp.Driver.DropTableIfExists(tx, table)
		if err != nil {
			return err
		}

		err = sp.Driver.CreateTableIfNotExists(tx, table)
		if err != nil {
			return err
		}

		return nil
	})
}

func (sp *SQLStreamPersister) Persist(name string, stream streams.Readable) error {
	sp.registeredTablesMu.RLock()
	table, ok := sp.registeredTables[name]
	sp.registeredTablesMu.RUnlock()

	if !ok {
		return fmt.Errorf("sqlpersistence: trying to persist unregistered table")
	}

	return sp.inTransaction(func(tx *sql.Tx) error {
		err := sp.Driver.PersistStream(tx, table, stream)
		if err != nil {
			return err
		}

		return nil
	})
}

func (sp *SQLStreamPersister) inTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := sp.db.Begin()
	if err != nil {
		return err
	}

	origErr := fn(tx)
	if origErr != nil {
		err = tx.Rollback()
		if err != nil {
			return err
		}

		return origErr
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

type Column struct {
	OriginalFieldName string
	ColumnName        string
	ColumnType        string
	PrimaryKey        bool
	ConvertFn         func(v interface{}) interface{}
}

func newColumn(originalFieldName, columnName string, primaryKey bool) *Column {
	return &Column{
		OriginalFieldName: originalFieldName,
		ColumnName:        columnName,
		PrimaryKey:        primaryKey,
	}
}

type Table struct {
	TableName string
	Columns   []*Column
}

func newTable(tableName string) *Table {
	return &Table{
		TableName: tableName,
		Columns:   []*Column{},
	}
}

func (t *Table) GetColumnNames() []string {
	columnNames := []string{}
	for _, c := range t.Columns {
		columnNames = append(columnNames, c.ColumnName)
	}
	return columnNames
}

func (t *Table) GetColumnValues(obj interface{}) []interface{} {
	columnValues := []interface{}{}
	if obj == nil {
		return columnValues
	}

	v := reflect.ValueOf(obj).Elem()
	for _, c := range t.Columns {
		fv := v.FieldByName(c.OriginalFieldName).Interface()
		if c.ConvertFn != nil {
			fv = c.ConvertFn(fv)
		}
		columnValues = append(columnValues, fv)
	}
	return columnValues
}