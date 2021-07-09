package archive

import (
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

type RawSQLMigration struct {
	migrator.MigrationBase
	sql string
}

func NewRawSQLMigration(sql string) *RawSQLMigration {
	return &RawSQLMigration{sql: sql}
}

func (m *RawSQLMigration) Sql(dialect migrator.Dialect) string {
	return m.sql
}

func InitDatabase(dbType string, connectionString string) (*xorm.Engine, error) {
	x, err := xorm.NewEngine(dbType, connectionString)
	x.SetColumnMapper(core.GonicMapper{})
	if err != nil {
		return nil, err
	}

	mig := migrator.NewMigrator(x)

	migrationLog := migrator.Table{
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

	mig.AddMigration("create migration_log table", migrator.NewAddTableMigration(migrationLog))

	archiveFile := migrator.Table{
		Name: "archive_file",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true},
			{Name: "created_at", Type: migrator.DB_DateTime},
		},
	}

	mig.AddMigration("create archive file table", migrator.NewAddTableMigration(archiveFile))

	githubEvent := migrator.Table{
		Name: "github_event",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt},
			{Name: "created_at", Type: migrator.DB_DateTime},
			{Name: "data", Type: migrator.DB_Text},
		},
	}

	mig.AddMigration("create github event table", migrator.NewAddTableMigration(githubEvent))

	githubEventPkey := NewRawSQLMigration("ALTER TABLE github_event ADD PRIMARY KEY (id)")

	mig.AddMigration("add primary key to github event table", githubEventPkey)

	return x, mig.Start()
}
