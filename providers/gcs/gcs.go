package gcs

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

// Gcs is a file storage provider for Google Cloud Storage
type Gcs struct {
	bucket string
	client *storage.BucketHandle
}

// New returns a new Gcs provider
func New(bucket *storage.BucketHandle) *Gcs {
	return &Gcs{client: bucket}
}

// OpenFileWriter opens a file for writing
func (g *Gcs) WriteFile(ctx context.Context, path string, r io.ReadSeeker) error {
	w := g.client.Object(path).NewWriter(ctx)
	defer w.Close()
	_, err := io.Copy(w, r)
	return err
}

// OpenFileReader opens a file for reading
func (g *Gcs) ReadFile(ctx context.Context, path string, w io.Writer) error {
	r, err := g.client.Object(path).NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	return err
}

// DeleteFile deletes a file
func (g *Gcs) DeleteFile(ctx context.Context, path string) error {
	return g.client.Object(path).Delete(ctx)
}
