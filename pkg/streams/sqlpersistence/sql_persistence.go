package sqlpersistence

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grafana/devtools/pkg/streams"
	"github.com/grafana/devtools/pkg/streams/log"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

type Driver interface {
	Init(logger log.Logger) error
	DropTableIfExists(tx *sql.Tx, persistedStream *Table) error
	CreateTableIfNotExists(tx *sql.Tx, persistedStream *Table) error
	PersistStream(tx *sql.Tx, persistedStream *Table, stream streams.Readable) (int64, error)
}

// Register makes a sql stream persister driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("Register called twice for driver " + name)
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
	DriverName         string
	ConnectionString   string
	Driver             Driver
	logger             log.Logger
	registeredTablesMu sync.RWMutex
	registeredTables   map[string]*Table
}

func Open(logger log.Logger, driverName, connectionString string) (*SQLStreamPersister, error) {
	driversMu.RLock()
	driveri, ok := drivers[driverName]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown driver %q (forgotten import?)", driverName)
	}

	err := driveri.Init(logger)
	if err != nil {
		return nil, err
	}

	sqlStreamPersister := SQLStreamPersister{
		logger:           logger.New("logger", "sql-persistence"),
		DriverName:       driverName,
		ConnectionString: connectionString,
		Driver:           driveri,
		registeredTables: map[string]*Table{},
	}

	db, err := sqlStreamPersister.connect()
	defer sqlStreamPersister.closeConnection(db)
	if err != nil {
		return nil, err
	}

	return &sqlStreamPersister, nil
}

func (sp *SQLStreamPersister) connect() (*sql.DB, error) {
	db, err := sql.Open(sp.DriverName, sp.ConnectionString)
	if err != nil {
		sp.logger.Error("failed to connect to database", "driver", sp.DriverName)
		return nil, err
	}

	if err = db.Ping(); err != nil {
		sp.logger.Error("failed to ping database", "driver", sp.DriverName)
		return nil, err
	}

	return db, nil
}

func (sp *SQLStreamPersister) closeConnection(db *sql.DB) {
	if db != nil {
		err := db.Close()
		if err != nil {
			sp.logger.Error("failed to close connection", "error", err)
		}
	}
}

func (sp *SQLStreamPersister) Register(name string, objTemplate interface{}) error {
	sp.logger.Debug("registering database table...", "tableName", name)

	table := newTable(name)
	t := reflect.TypeOf(objTemplate).Elem()
	t.NumField()
	setColumnDataType := func(t reflect.Type, c *Column) {
		switch t.Kind() {
		case reflect.TypeOf(time.Time{}).Kind():
			c.Type = ColumnTypeInteger
			c.Length = 32
			c.ConvertFn = func(v interface{}) interface{} {
				return v.(time.Time).Unix()
			}
		case reflect.String:
			c.Type = ColumnTypeString

			if c.Length == 0 {
				c.Length = 256
			}

		case reflect.Float64:
			c.Type = ColumnTypeFloat
			c.Length = 64
		case reflect.Float32:
			c.Type = ColumnTypeFloat
			c.Length = 32
		case reflect.Int, reflect.Int32:
			c.Type = ColumnTypeInteger
			c.Length = 32
		case reflect.Int64:
			c.Type = ColumnTypeInteger
			c.Length = 64
		case reflect.Bool:
			c.Type = ColumnTypeBoolean
		}
	}

	for n := 0; n < t.NumField(); n++ {
		f := t.Field(n)
		if !reflect.ValueOf(objTemplate).Elem().Field(n).CanSet() {
			continue
		}

		c := newColumn(f.Name, strings.ToLower(f.Name))

		tag := f.Tag.Get("persist")
		if tag != "" {
			parts := strings.Split(tag, ",")
			if len(parts) > 0 && parts[0] == "-" {
				continue
			}

			if len(parts) > 0 && parts[0] != "" {
				c.Name = strings.ToLower(parts[0])
			}

			if len(parts) > 1 {
				if strings.Contains(parts[1], "primarykey") {
					c.IsPrimaryKey = true
				}

				if strings.Contains(parts[1], "not null") {
					c.IsNullable = false
				} else if strings.Contains(parts[1], "null") {
					c.IsNullable = true
				}

				if strings.Contains(parts[1], "unicode") {
					c.IsUnicode = true
				}

				lengthIndex := strings.Index(parts[1], "length(")
				lastLengthIndex := strings.LastIndex(parts[1], ")")
				if lengthIndex != -1 && lastLengthIndex != -1 {
					lengthStr := parts[1][lengthIndex+7 : lastLengthIndex]
					if length, err := strconv.Atoi(lengthStr); err != nil {
						sp.logger.Error("failed to parse column length to integer", "input", lengthStr)
					} else {
						c.Length = length
					}
				}
			}
		}

		setColumnDataType(f.Type, c)
		table.Columns = append(table.Columns, c)
	}

	sp.registeredTablesMu.Lock()
	sp.registeredTables[name] = table
	sp.registeredTablesMu.Unlock()

	sp.logger.Debug("database table registered", "tableName", name, "columns", table.GetColumnNames())

	db, err := sp.connect()
	defer sp.closeConnection(db)
	if err != nil {
		return err
	}

	return sp.inTransaction(db, func(tx *sql.Tx) error {
		sp.logger.Debug("dropping table", "tableName", table.TableName)
		err := sp.Driver.DropTableIfExists(tx, table)
		if err != nil {
			return err
		}

		sp.logger.Debug("creating table", "tableName", table.TableName)
		err = sp.Driver.CreateTableIfNotExists(tx, table)
		if err != nil {
			return err
		}

		return nil
	})
}

func (sp *SQLStreamPersister) Persist(name string, stream streams.Readable) error {
	start := time.Now()
	sp.logger.Debug("persisting stream to database table...", "tableName", name)

	sp.registeredTablesMu.RLock()
	table, ok := sp.registeredTables[name]
	sp.registeredTablesMu.RUnlock()

	if !ok {
		return fmt.Errorf("trying to persist unregistered table")
	}

	db, err := sp.connect()
	defer sp.closeConnection(db)
	if err != nil {
		return err
	}

	return sp.inTransaction(db, func(tx *sql.Tx) error {
		rowsAffected, err := sp.Driver.PersistStream(tx, table, stream)
		if err != nil {
			sp.logger.Error("failed to persist stream to database", "table", name, "took", time.Since(start))
			return err
		}

		sp.logger.Debug("stream persisted to database", "table", name, "took", time.Since(start), "rowsAffected", rowsAffected)

		return nil
	})
}

func (sp *SQLStreamPersister) inTransaction(db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
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

type ColumnType int

const (
	ColumnTypeUnknown ColumnType = 1 << iota
	ColumnTypeInteger
	ColumnTypeFloat
	ColumnTypeBoolean
	ColumnTypeString
)

func (ct ColumnType) String() string {
	names := map[int]string{
		int(ColumnTypeUnknown): "unknown",
		int(ColumnTypeInteger): "integer",
		int(ColumnTypeFloat):   "float",
		int(ColumnTypeString):  "string",
		int(ColumnTypeBoolean): "boolean",
	}
	return names[int(ct)]
}

type Column struct {
	OriginalFieldName string
	Name              string
	Type              ColumnType
	Length            int
	DatabaseType      string
	IsPrimaryKey      bool
	IsNullable        bool
	IsUnicode         bool
	ConvertFn         func(v interface{}) interface{}
}

func newColumn(originalFieldName, name string) *Column {
	return &Column{
		OriginalFieldName: originalFieldName,
		Name:              name,
		Type:              ColumnTypeUnknown,
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
		columnNames = append(columnNames, c.Name)
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
