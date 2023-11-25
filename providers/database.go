//go:generate mockgen -destination=./mocks/database.go -package=mocks . Database

package providers

import (
	"context"

	"github.com/spf13/viper"
)

// Database is an interface that represents a database client. It is used to run migrations and close the connection.
// Type casting is required to use the client.
type Database interface {
	// Migrate runs the database migrations
	Migrate(ctx context.Context) error
	// Close closes the database connection
	Close() error
}

// DatabaseProvider is a function that returns a Database
type DatabaseProvider func(ctx context.Context, config *viper.Viper) (Database, error)
