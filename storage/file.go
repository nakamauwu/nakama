package storage

import (
	"io"
	"time"
)

// File contents and info.
type File struct {
	io.ReadSeekCloser

	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}
