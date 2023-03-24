package nakama

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

const (
	S3BucketAvatars = "avatars"
)

type s3StoreObject struct {
	File        io.Reader
	Bucket      string
	Name        string
	Size        uint64
	ContentType string
}

type s3RemoveObject struct {
	Bucket string
	Name   string
}

func (svc *Service) s3StoreObject(ctx context.Context, in s3StoreObject) error {
	_, err := svc.S3.PutObject(ctx, in.Bucket, in.Name, in.File, int64(in.Size), minio.PutObjectOptions{
		ContentType: in.ContentType,
	})
	return err
}

func (svc *Service) s3RemoveObject(ctx context.Context, in s3RemoveObject) error {
	return svc.S3.RemoveObject(ctx, in.Bucket, in.Name, minio.RemoveObjectOptions{})
}
