//go:generate mockgen -destination=./mocks/cache.go -package=mocks . Cache

package providers

import (
	"context"
	"time"

	"github.com/spf13/viper"
)

// Cache is an interface for caching data
type Cache interface {
	// Get returns a value from the cache by key
	Get(ctx context.Context, key string) (string, error)
	// Set sets a value in the cache by key
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	// Delete deletes a value from the cache by key
	Delete(ctx context.Context, key string) error
	// Lock locks a key in the cache for a given ttl. It returns true if the lock was acquired, false otherwise.
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Unlock unlocks a key in the cache
	Unlock(ctx context.Context, key string, ttl time.Duration) error
	// Once runs a function once for a given key. It returns true if the function was run, false otherwise.
	Once(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error) (bool, error)
}

// CacheProvider is a function that returns a Cache
type CacheProvider func(ctx context.Context, config *viper.Viper) (Cache, error)
