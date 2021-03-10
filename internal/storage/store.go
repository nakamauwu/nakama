package storage

import (
	"context"
	"errors"
)

// ErrNotFound denotes that the object does not exists.
var ErrNotFound = errors.New("not found")

// Store interface.
type Store interface {
	Store(ctx context.Context, name string, data []byte, opts ...func(*StoreOpts)) (err error)
	Open(ctx context.Context, name string) (f *File, err error)
	Delete(ctx context.Context, name string) (err error)
}
