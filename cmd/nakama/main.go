package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nakamauwu/nakama"
	"github.com/nakamauwu/nakama/web"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
	})
	if err := run(logger); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

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

	s3Prefix := s3Prefix(s3Secure, s3Endpoint)
	store := nakama.NewStore(pool, s3Prefix)

	s3, err := minio.New(s3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
		Secure: s3Secure,
	})
	if err != nil {
		return fmt.Errorf("s3: %w", err)
	}

	if err := nakama.EnsureS3Buckets(ctx, s3); err != nil {
		return fmt.Errorf("s3: ensure buckets: %w", err)
	}

	svc := &nakama.Service{
		Store:       store,
		S3:          s3,
		Logger:      logger.With("component", "service"),
		S3Prefix:    s3Prefix,
		BaseContext: func() context.Context { return ctx },
	}

	handler := &web.Handler{
		Logger:     logger.With("component", "web"),
		Service:    svc,
		SessionKey: []byte(sessionKey),
	}

	srv := &http.Server{
		Handler:     handler,
		Addr:        addr,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	defer srv.Close()

	go func() {
		<-ctx.Done()
		fmt.Println()
		logger.Info("shutting down server")

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			logger.Error("shutdown server", "err", err)
			os.Exit(1)
		}
	}()

	logger.Info("starting server", "addr", srv.Addr)

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	svc.Wait()

	return nil
}

func s3Prefix(useSSL bool, endpoint string) string {
	out := "http"
	if useSSL {
		out += "s"
	}
	return out + "://" + endpoint + "/"
}
