package archive

import (
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

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
			{Name: "type", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "repo_id", Type: migrator.DB_BigInt},
			{Name: "created_at", Type: migrator.DB_DateTime},
			{Name: "actor", Type: migrator.DB_Text},
			{Name: "payload", Type: migrator.DB_Text},
		},
	}

	mig.AddMigration("create github event table", migrator.NewAddTableMigration(githubEvent))

	aggregationStats := migrator.Table{
		Name: "aggregated_stats",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt},
			{Name: "issue_count", Type: migrator.DB_BigInt},
			{Name: "issue_comment_count", Type: migrator.DB_BigInt},
			{Name: "pr_count", Type: migrator.DB_BigInt},
			{Name: "watcher_count", Type: migrator.DB_BigInt},
		},
	}

	mig.AddMigration("create aggregation stats table", migrator.NewAddTableMigration(aggregationStats))

	return x, mig.Start()
}
