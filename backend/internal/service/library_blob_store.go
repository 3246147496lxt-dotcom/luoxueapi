package service

import (
	"context"
	"io"
	"time"
)

// LibraryBlobStore persists private library objects without exposing a public
// URL. Callers retain ownership of the input reader; callers must close streams
// returned by Open.
type LibraryBlobStore interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// LibraryBlobOrphanCleaner is optional because some private object stores may
// not provide safe listing semantics. Implementations must only delete
// canonical objects older than olderThan and absent from referenced.
type LibraryBlobOrphanCleaner interface {
	CleanupOrphans(ctx context.Context, referenced map[string]struct{}, olderThan time.Time, limit int) (int, error)
}
