package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/policy"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/nakamauwu/nakama"
	"github.com/nakamauwu/nakama/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	var (
		addr        string
		sqlAddr     string
		sessionKey  string
		s3Endpoint  string
		s3AccessKey string
		s3SecretKey string
		s3Secure    bool
	)

	fs := flag.NewFlagSet("nakama", flag.ExitOnError)
	fs.StringVar(&addr, "addr", ":4000", "HTTP service address")
	fs.StringVar(&sqlAddr, "sql-addr", "postgresql://root@127.0.0.1:26257/defaultdb?sslmode=disable", "SQL address")
	fs.StringVar(&sessionKey, "session-key", "secretkeyyoushouldnotcommit", "Session key used to authenticate and encrypt cookies")
	fs.StringVar(&s3Endpoint, "s3-endpoint", "localhost:9000", "S3 endpoint")
	fs.StringVar(&s3AccessKey, "s3-access-key", "minioadmin", "S3 access key")
	fs.StringVar(&s3SecretKey, "s3-secret-key", "minioadmin", "S3 secret key")
	fs.BoolVar(&s3Secure, "s3-secure", false, "Enable S3 SSL")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	pool, err := pgxpool.New(ctx, sqlAddr)
	if err != nil {
		return fmt.Errorf("open pool: %w", err)
	}

	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if err := nakama.MigrateSQL(ctx, pool); err != nil {
		return fmt.Errorf("migrate sql: %w", err)
	}

	avatarsPrefix := avatarsPrefix(s3Secure, s3Endpoint)

	store := nakama.NewStore(pool)
	store.AvatarScanFunc = nakama.MakePrefixedNullStringScanner(avatarsPrefix)

	s3, err := minio.New(s3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
		Secure: s3Secure,
	})
	if err != nil {
		return fmt.Errorf("s3: %w", err)
	}

	if err := ensureS3Buckets(ctx, s3); err != nil {
		return fmt.Errorf("s3: ensure buckets: %w", err)
	}

	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Llongfile)
	svc := &nakama.Service{
		Store:         store,
		S3:            s3,
		Logger:        logger,
		AvatarsPrefix: avatarsPrefix,
		BaseContext:   func() context.Context { return ctx },
	}

	handler := &web.Handler{
		Logger:     logger,
		Service:    svc,
		SessionKey: []byte(sessionKey),
	}

	srv := &http.Server{
		Handler: handler,
		Addr:    addr,
	}

	defer srv.Close()

	logger.Printf("listening on %s", addr)

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func ensureS3Buckets(ctx context.Context, s3 *minio.Client) error {
	for _, bucket := range [...]string{nakama.S3BucketAvatars} {
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

		policy, err := s3MakeAllowGetObjectPolicy(bucket)
		if err != nil {
			return err
		}

		err = s3.SetBucketPolicy(ctx, bucket, policy)
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

func avatarsPrefix(useSSL bool, endpoint string) string {
	out := "http"
	if useSSL {
		out += "s"
	}
	return out + "://" + endpoint + "/" + nakama.S3BucketAvatars + "/"
}
