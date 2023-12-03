package filesys

import (
	"context"
	"io"
	"os"
)

// Filesys is a file storage provider for the local filesystem
type Filesys struct {
	baseDir string
}

// New returns a new Filesys provider
func New(baseDir string) *Filesys {
	return &Filesys{
		baseDir: baseDir,
	}
}

// WriteFile writes a file to the storage provider
func (f Filesys) WriteFile(ctx context.Context, path string, r io.ReadSeeker) error {
	path = f.baseDir + path
	fl, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fl.Close()
	_, err = io.Copy(fl, r)
	if err != nil {
		return err
	}
	return nil
}

// ReadFile reads a file from the storage provider
func (f Filesys) ReadFile(ctx context.Context, path string, w io.Writer) error {
	fl, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fl.Close()
	_, err = io.Copy(w, fl)
	if err != nil {
		return err
	}
	return nil
}

// DeleteFile deletes a file
func (f Filesys) DeleteFile(ctx context.Context, path string) error {
	return os.Remove(path)
}
