package s3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nakamauwu/nakama/storage"
)

// Store must call Init.
type Store struct {
	client *minio.Client

	Secure     bool
	Endpoint   string
	Region     string
	AccessKey  string
	SecretKey  string
	BucketList []string
}

func (s *Store) Setup(ctx context.Context) error {
	var err error
	s.client, err = minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
		Secure: s.Secure,
		Region: s.Region,
	})
	if err != nil {
		return fmt.Errorf("could not create minio client: %w", err)
	}

	for _, bucket := range s.BucketList {
		err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{
			Region: s.Region,
		})
		if err == nil {
			continue
		}

		exists, errExists := s.client.BucketExists(ctx, bucket)
		if errExists != nil {
			return fmt.Errorf("could not check bucket %q existence: %w", bucket, errExists)
		}

		if exists {
			continue
		}

		return fmt.Errorf("could not create bucket %q: %w", bucket, err)
	}

	return nil
}

// Store a file.
func (s *Store) Store(ctx context.Context, bucket, name string, data []byte, opts ...func(*storage.StoreOpts)) error {
	var options storage.StoreOpts
	for _, o := range opts {
		o(&options)
	}

	r := bytes.NewReader(data)
	size := int64(len(data))
	_, err := s.client.PutObject(ctx, bucket, name, r, size, minio.PutObjectOptions{
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
func (s *Store) Open(ctx context.Context, bucket, name string) (*storage.File, error) {
	obj, err := s.client.GetObject(ctx, bucket, name, minio.GetObjectOptions{})
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
func (s *Store) Delete(ctx context.Context, bucket, name string) error {
	err := s.client.RemoveObject(ctx, bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("could not delete object: %w", err)
	}

	return nil
}
