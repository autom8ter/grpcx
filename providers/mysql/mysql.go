package mysql

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"

	"github.com/autom8ter/grpcx/internal/utils"
	"github.com/autom8ter/grpcx/providers"
)

type mysql struct {
	conn    *sql.DB
	migrate *migrate.Migrate
}

// New returns a new mysql database client
func New(ctx context.Context, connectionString, migrationsPath string) (providers.Database, error) {
	conn, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, utils.WrapError(err, "failed to open database connection")
	}
	db := &mysql{conn: conn}
	m, err := migrate.New(migrationsPath, connectionString)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create migration")
	}
	db.migrate = m
	return db, nil
}

// Migrate runs the database migrations
func (s mysql) Migrate(ctx context.Context) error {
	if s.migrate == nil {
		return nil
	}
	return utils.WrapError(s.migrate.Up(), "")
}

// Close closes the database connection
func (s mysql) Close() error {
	return s.conn.Close()
}
