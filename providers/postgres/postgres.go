package postgres

import (
	"context"
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"

	"github.com/autom8ter/grpcx/internal/utils"
	"github.com/autom8ter/grpcx/providers"
)

type postgres struct {
	conn    *sql.DB
	migrate *migrate.Migrate
}

// New returns a new postgres database client
func New(ctx context.Context, connectionString, migrationsPath string) (providers.Database, error) {
	conn, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, utils.WrapError(err, "failed to open database connection")
	}
	db := &postgres{conn: conn}
	m, err := migrate.New(migrationsPath, connectionString)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create migration")
	}
	db.migrate = m
	return db, nil
}

// Migrate runs the database migrations
func (s postgres) Migrate(ctx context.Context) error {
	if s.migrate == nil {
		return nil
	}
	return utils.WrapError(s.migrate.Up(), "")
}

// Close closes the database connection
func (s postgres) Close() error {
	return s.conn.Close()
}
