//go:generate mockgen -destination=./mocks/storage.go -package=mocks . Storage

package providers

import (
	"context"
	"io"
)

// Storage is an interface for a file storage provider
type Storage interface {
	// WriteFile writes a file to the storage provider
	WriteFile(ctx context.Context, path string, r io.ReadSeeker) error
	// ReadFile reads a file from the storage provider
	ReadFile(ctx context.Context, path string, w io.Writer) error
	// DeleteFile deletes a file
	DeleteFile(ctx context.Context, path string) error
}
