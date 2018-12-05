package streamprojections

import (
	"reflect"
	"strings"
	"time"

	"github.com/grafana/github-repo-metrics/pkg/streams/pkg/streams"
)

type StreamPersister interface {
	Register(name string, objTemplate interface{})
	Persist(name string, stream streams.Readable)
}

type PersistedStreamColumn struct {
	OriginalFieldName string
	ColumnName        string
	ColumnType        string
	PrimaryKey        bool
	ConvertFn         func(v interface{}) interface{}
}

func newPersistedStreamColumn(originalFieldName, columnName string, primaryKey bool) *PersistedStreamColumn {
	return &PersistedStreamColumn{
		OriginalFieldName: originalFieldName,
		ColumnName:        columnName,
		PrimaryKey:        primaryKey,
	}
}

type PersistedStream struct {
	Name    string
	Columns []*PersistedStreamColumn
}

func newPersistedStream(name string) *PersistedStream {
	return &PersistedStream{
		Name:    name,
		Columns: []*PersistedStreamColumn{},
	}
}

func (ps *PersistedStream) GetColumnNames() []string {
	columnNames := []string{}
	for _, c := range ps.Columns {
		columnNames = append(columnNames, c.ColumnName)
	}
	return columnNames
}

func (ps *PersistedStream) GetColumnValues(obj interface{}) []interface{} {
	columnValues := []interface{}{}
	v := reflect.ValueOf(obj).Elem()
	for _, c := range ps.Columns {
		fv := v.FieldByName(c.OriginalFieldName).Interface()
		if c.ConvertFn != nil {
			fv = c.ConvertFn(fv)
		}
		columnValues = append(columnValues, fv)
	}
	return columnValues
}

type StreamPersisterBase struct {
	StreamsToPersist map[string]*PersistedStream
}

func NewStreamPersisterBase() *StreamPersisterBase {
	return &StreamPersisterBase{
		StreamsToPersist: map[string]*PersistedStream{},
	}
}

func (sp *StreamPersisterBase) Register(name string, objTemplate interface{}) {
	ps := newPersistedStream(name)
	t := reflect.TypeOf(objTemplate).Elem()
	t.NumField()
	setColumnDataType := func(t reflect.Type, c *PersistedStreamColumn) {
		switch t.Kind() {
		case reflect.TypeOf(time.Time{}).Kind():
			c.ColumnType = "integer"
			c.ConvertFn = func(v interface{}) interface{} {
				return v.(time.Time).Unix()
			}
		case reflect.String:
			c.ColumnType = "text"
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

		c := newPersistedStreamColumn(f.Name, columnName, primaryKey)
		setColumnDataType(f.Type, c)
		ps.Columns = append(ps.Columns, c)
	}
	sp.StreamsToPersist[name] = ps
}

func (sp *StreamPersisterBase) Persist(name string, stream streams.Readable) {
	for _ = range stream {

	}
}
