package storage

import (
	"context"
	"errors"
)

// ErrNotFound denotes that the object does not exists.
var ErrNotFound = errors.New("not found")

// Store interface.
type Store interface {
	Store(ctx context.Context, bucket, name string, data []byte, opts ...func(*StoreOpts)) (err error)
	Open(ctx context.Context, bucket, name string) (f *File, err error)
	Delete(ctx context.Context, bucket, name string) (err error)
}
