package nakama

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/policy"
	"github.com/minio/minio-go/v7/pkg/set"
)

const (
	S3BucketAvatars = "avatars"
	S3BucketMedia   = "media"
)

var AllS3Buckets = []string{S3BucketAvatars, S3BucketMedia}

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

func EnsureS3Buckets(ctx context.Context, s3 *minio.Client) error {
	for _, bucket := range AllS3Buckets {
		ok, err := s3.BucketExists(ctx, bucket)
		if err != nil {
			return err
		}

		if ok {
			continue
		}

		err = s3.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}

		pol, err := s3MakeAllowGetObjectPolicy(bucket)
		if err != nil {
			return err
		}

		err = s3.SetBucketPolicy(ctx, bucket, pol)
		if err != nil {
			return err
		}
	}

	return nil
}

func s3MakeAllowGetObjectPolicy(bucket string) (string, error) {
	p := policy.BucketAccessPolicy{
		Version: "2012-10-17", // See: https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_version.html
		Statements: []policy.Statement{{
			Effect:    "Allow",
			Actions:   set.CreateStringSet("s3:GetObject"),
			Principal: policy.User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet(fmt.Sprintf("arn:aws:s3:::%s/*", bucket)),
		}},
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}
