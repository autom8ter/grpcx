package sqlite

import (
	"context"
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3"

	"github.com/autom8ter/grpcx/internal/utils"
	"github.com/autom8ter/grpcx/providers"
)

type sqlite struct {
	conn    *sql.DB
	migrate *migrate.Migrate
}

func (s sqlite) Migrate(ctx context.Context) error {
	if s.migrate == nil {
		return nil
	}
	return utils.WrapError(s.migrate.Up(), "")
}

func (s sqlite) Close() error {
	return s.conn.Close()
}

// New returns a new sqlite database client
func New(ctx context.Context, connectionString, migrationsPath string) (providers.Database, error) {
	conn, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, utils.WrapError(err, "failed to open database connection")
	}
	db := &sqlite{conn: conn}
	m, err := migrate.New(migrationsPath, connectionString)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create migration")
	}
	db.migrate = m
	return db, nil
}
