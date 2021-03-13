package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	natslib "github.com/nats-io/nats.go"
	"github.com/nicolasparada/nakama/internal/handler"
	"github.com/nicolasparada/nakama/internal/mailing"
	"github.com/nicolasparada/nakama/internal/pubsub/nats"
	"github.com/nicolasparada/nakama/internal/service"
	"github.com/nicolasparada/nakama/internal/storage"
	"github.com/nicolasparada/nakama/internal/storage/fs"
	"github.com/nicolasparada/nakama/internal/storage/s3"
)

//go:embed schema.sql
var schema string

func main() {
	_ = godotenv.Load()
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var (
		port, _                   = strconv.Atoi(env("PORT", "3000"))
		originStr                 = env("ORIGIN", fmt.Sprintf("http://localhost:%d", port))
		dbURL                     = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable")
		execSchema, _             = strconv.ParseBool(env("EXEC_SCHEMA", "false"))
		tokenKey                  = env("TOKEN_KEY", "supersecretkeyyoushouldnotcommit")
		natsURL                   = env("NATS_URL", natslib.DefaultURL)
		sendgridAPIKey            = os.Getenv("SENDGRID_API_KEY")
		smtpHost                  = env("SMTP_HOST", "smtp.mailtrap.io")
		smtpPort, _               = strconv.Atoi(env("SMTP_PORT", "25"))
		smtpUsername              = os.Getenv("SMTP_USERNAME")
		smtpPassword              = os.Getenv("SMTP_PASSWORD")
		enableStaticFilesCache, _ = strconv.ParseBool(env("STATIC_CACHE", "false"))
		embedStaticFiles, _       = strconv.ParseBool(env("EMBED_STATIC", "false"))
		s3Endpoint                = os.Getenv("S3_ENDPOINT")
		s3Region                  = os.Getenv("S3_REGION")
		s3Bucket                  = env("S3_BUCKET", "avatars")
		s3AccessKey               = os.Getenv("S3_ACCESS_KEY")
		s3SecretKey               = os.Getenv("S3_SECRET_KEY")
	)
	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Println("\nDon't forget to set TOKEN_KEY, and SENDGRID_API_KEY or SMTP_USERNAME and SMTP_PASSWORD for real usage.")
	}
	flag.IntVar(&port, "port", port, "Port in which this server will run")
	flag.StringVar(&originStr, "origin", originStr, "URL origin for this service")
	flag.StringVar(&dbURL, "db", dbURL, "Database URL")
	flag.BoolVar(&execSchema, "exec-schema", execSchema, "Execute database schema")
	flag.StringVar(&natsURL, "nats", natsURL, "NATS URL")
	flag.StringVar(&smtpHost, "smtp-host", smtpHost, "SMTP server host")
	flag.IntVar(&smtpPort, "smtp-port", smtpPort, "SMTP server port")
	flag.BoolVar(&enableStaticFilesCache, "static-cache", enableStaticFilesCache, "Enable static files cache")
	flag.BoolVar(&embedStaticFiles, "embed-static", embedStaticFiles, "Embed static files")
	flag.Parse()

	origin, err := url.Parse(originStr)
	if err != nil || !origin.IsAbs() {
		return errors.New("invalid url origin")
	}

	if i, err := strconv.Atoi(origin.Port()); err == nil {
		port = i
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("could not open db connection: %w", err)
	}

	defer db.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("could not ping to db: %w", err)
	}

	if execSchema {
		_, err := db.ExecContext(ctx, schema)
		if err != nil {
			return fmt.Errorf("could not run schema: %w", err)
		}
	}

	natsConn, err := natslib.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("could not connect to NATS server: %w", err)
	}

	pubsub := &nats.PubSub{Conn: natsConn}

	var sender mailing.Sender
	sendFrom := "no-reply@" + origin.Hostname()
	if sendgridAPIKey != "" {
		log.Println("using sendgrid mailing implementation")
		sender = mailing.NewSendgridSender(sendFrom, sendgridAPIKey)
	} else if smtpUsername != "" && smtpPassword != "" {
		log.Println("using smtp mailing implementation")
		sender = mailing.NewSMTPSender(
			sendFrom,
			smtpHost,
			smtpPort,
			smtpUsername,
			smtpPassword,
		)
	} else {
		log.Println("using log mailing implementation")
		sender = mailing.NewLogSender(
			sendFrom,
			&logWrapper{Logger: log.New(os.Stdout, "mailing ", log.LstdFlags)},
		)
	}

	var store storage.Store
	s3Enabled := s3Endpoint != "" && s3AccessKey != "" && s3SecretKey != ""
	if s3Enabled {
		log.Println("using s3 store implementation")
		store = &s3.Store{
			Endpoint:  s3Endpoint,
			Region:    s3Region,
			Bucket:    s3Bucket,
			AccessKey: s3AccessKey,
			SecretKey: s3SecretKey,
		}
	} else {
		log.Println("using os file system store implementation")
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current working directory: %w", err)
		}

		store = &fs.Store{Root: filepath.Join(wd, "web", "static", "img", "avatars")}
	}

	svc := &service.Service{
		DB:       db,
		Sender:   sender,
		Origin:   origin,
		TokenKey: tokenKey,
		PubSub:   pubsub,
		Store:    store,
	}

	go svc.RunBackgroundJobs(ctx)

	serveAvatars := !s3Enabled
	h := handler.New(ctx, svc, store, enableStaticFilesCache, embedStaticFiles, serveAvatars)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           h,
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       time.Second * 15,
	}

	errs := make(chan error, 1)
	go func() {
		<-ctx.Done()

		fmt.Println()
		log.Println("gracefully shutting down...")
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			errs <- fmt.Errorf("could not shutdown server: %w", err)
		}

		log.Println("ok")

		errs <- nil
	}()

	log.Printf("accepting connections on port %d\n", port)
	log.Printf("starting server at %s\n", origin)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		close(errs)
		return fmt.Errorf("could not listen and serve: %w", err)
	}

	return <-errs
}

func env(key, fallbackValue string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return fallbackValue
	}
	return s
}

type logWrapper struct {
	Logger *log.Logger
}

func (l *logWrapper) Log(args ...interface{}) { l.Logger.Println(args...) }

func (l *logWrapper) Logf(format string, args ...interface{}) { l.Logger.Printf(format, args...) }
