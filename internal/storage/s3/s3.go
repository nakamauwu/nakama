package s3

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nicolasparada/nakama/internal/storage"
)

// Store must call Init.
type Store struct {
	client *minio.Client
	once   sync.Once

	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

func (s *Store) init(ctx context.Context) (err error) {
	s.client, err = minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
		Secure: strings.HasPrefix(s.Endpoint, "https:"),
	})
	if err != nil {
		return fmt.Errorf("could not create minio client: %w", err)
	}

	err = s.client.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{
		Region: s.Region,
	})
	if err != nil {
		exists, errExists := s.client.BucketExists(ctx, s.Bucket)
		if errExists != nil {
			return fmt.Errorf("could not check bucket %q existence: %w", s.Bucket, errExists)
		}

		if exists {
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("could not create bucket %q: %w", s.Bucket, err)
	}

	return nil
}

// Store a file.
func (s *Store) Store(ctx context.Context, name string, data []byte, opts ...func(*storage.StoreOpts)) (err error) {
	s.once.Do(func() {
		err = s.init(ctx)
	})
	if err != nil {
		return fmt.Errorf("could not init minio client: %w", err)
	}

	var options storage.StoreOpts
	for _, o := range opts {
		o(&options)
	}

	r := bytes.NewReader(data)
	size := int64(len(data))
	_, err = s.client.PutObject(ctx, s.Bucket, name, r, size, minio.PutObjectOptions{
		ContentType:     options.ContentType,
		ContentEncoding: options.ContentEncoding,
		CacheControl:    options.CacheControl,
	})
	if err != nil {
		return fmt.Errorf("could not put object: %w", err)
	}

	return nil
}

// Open a file.
func (s *Store) Open(ctx context.Context, name string) (f *storage.File, err error) {
	s.once.Do(func() {
		err = s.init(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("could not init minio client: %w", err)
	}

	obj, err := s.client.GetObject(ctx, s.Bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get object: %w", err)
	}

	stat, err := obj.Stat()
	if err != nil {
		if e := minio.ToErrorResponse(err); e.Code == "NoSuchKey" {
			return nil, storage.ErrNotFound
		}

		return nil, fmt.Errorf("could not stat %q (err type: %T): %w", name, err, err)
	}

	return &storage.File{
		ReadSeekCloser: obj,
		Size:           stat.Size,
		ContentType:    stat.ContentType,
		ETag:           stat.ETag,
		LastModified:   stat.LastModified,
	}, nil
}

// Delete a file.
func (s *Store) Delete(ctx context.Context, name string) (err error) {
	s.once.Do(func() {
		err = s.init(ctx)
	})
	if err != nil {
		return fmt.Errorf("could not init minio client: %w", err)
	}

	err = s.client.RemoveObject(ctx, s.Bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("could not delete object: %w", err)
	}

	return nil
}
