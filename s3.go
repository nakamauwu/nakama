package nakama

import (
	"context"
	"io"
	"strconv"

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
	Width       uint
	Height      uint
}

type s3RemoveObject struct {
	Bucket string
	Name   string
}

func (svc *Service) s3StoreObject(ctx context.Context, in s3StoreObject) error {
	usr, _ := UserFromContext(ctx)
	_, err := svc.S3.PutObject(ctx, string(in.Bucket), in.Name, in.File, int64(in.Size), minio.PutObjectOptions{
		ContentType: in.ContentType,
		UserMetadata: map[string]string{
			"width":  strconv.FormatUint(uint64(in.Width), 10),
			"height": strconv.FormatUint(uint64(in.Height), 10),
		},
		UserTags: map[string]string{
			"user_id": usr.ID,
		},
	})
	return err
}

func (svc *Service) s3RemoveObject(ctx context.Context, in s3RemoveObject) error {
	return svc.S3.RemoveObject(ctx, in.Bucket, in.Name, minio.RemoveObjectOptions{})
}
