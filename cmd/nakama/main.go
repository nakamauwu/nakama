package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nakamauwu/nakama"
	"github.com/nakamauwu/nakama/db"
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
	fs.StringVar(&sessionKey, "session-key", "secretkeyyoushouldnotcommit", "Session key")
	fs.StringVar(&s3Endpoint, "s3-endpoint", "localhost:9000", "S3 endpoint")
	fs.StringVar(&s3AccessKey, "s3-access-key", "minioadmin", "S3 access key")
	fs.StringVar(&s3SecretKey, "s3-secret-key", "minioadmin", "S3 secret key")
	fs.BoolVar(&s3Secure, "s3-secure", false, "Enable S3 SSL")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	pool, err := sql.Open("postgres", sqlAddr)
	if err != nil {
		return fmt.Errorf("open pool: %w", err)
	}

	defer pool.Close()

	if err := pool.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if err := nakama.MigrateSQL(ctx, pool); err != nil {
		return fmt.Errorf("migrate sql: %w", err)
	}

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
		Logger:        logger,
		DB:            db.New(pool),
		S3:            s3,
		AvatarsPrefix: avatarsPrefix(s3Secure, s3Endpoint),
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
	}

	return nil
}

func avatarsPrefix(useSSL bool, endpoint string) string {
	out := "http"
	if useSSL {
		out += "s"
	}
	return out + "://" + endpoint + "/" + nakama.S3BucketAvatars + "/"
}
