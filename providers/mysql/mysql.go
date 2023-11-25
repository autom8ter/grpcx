package mysql

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/palantir/stacktrace"
	"github.com/spf13/viper"

	"github.com/autom8ter/grpcx/providers"
)

type mysql struct {
	conn    *sql.DB
	migrate *migrate.Migrate
}

// New returns a new Database client
func New(conn *sql.DB, migrate *migrate.Migrate) providers.Database {
	return &mysql{
		conn:    conn,
		migrate: migrate,
	}
}

// Migrate runs the database migrations
func (s mysql) Migrate(ctx context.Context) error {
	if s.migrate == nil {
		return nil
	}
	return stacktrace.Propagate(s.migrate.Up(), "")
}

// Close closes the database connection
func (s mysql) Close() error {
	return s.conn.Close()
}

// Provider is a function that returns a Database client
// The config key "database.connection_string" must be set to the sqlite connection string
// If the config key "database.migrations_source" is set, the database will use
// github.com/golang-migrate/migrate to run migrations from the source set in the config key
func Provider(ctx context.Context, cfg *viper.Viper) (providers.Database, error) {
	if cfg.Get("database") == nil {
		return nil, stacktrace.NewError("config key 'database' not found")
	}
	if cfg.Get("database.connection_string") == nil {
		return nil, stacktrace.NewError("config key 'database.connection_string' not found")
	}
	conn, err := sql.Open("sqlite3", cfg.GetString("database.connection_string"))
	if err != nil {
		return nil, stacktrace.Propagate(err, "failed to open database connection")
	}
	db := &mysql{conn: conn}
	if path := cfg.GetString("database.migrations_source"); path != "" {
		m, err := migrate.New(path, cfg.GetString("database.connection_string"))
		if err != nil {
			return nil, stacktrace.Propagate(err, "failed to create migration")
		}
		db.migrate = m
	}
	return db, nil
}
