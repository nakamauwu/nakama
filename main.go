package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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
)

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
		tokenKey                  = env("TOKEN_KEY", "supersecretkeyyoushouldnotcommit")
		natsURL                   = env("NATS_URL", natslib.DefaultURL)
		smtpHost                  = env("SMTP_HOST", "smtp.mailtrap.io")
		smtpPort, _               = strconv.Atoi(env("SMTP_PORT", "25"))
		smtpUsername              = os.Getenv("SMTP_USERNAME")
		smtpPassword              = os.Getenv("SMTP_PASSWORD")
		enableStaticFilesCache, _ = strconv.ParseBool(env("ENABLE_STATIC_FILES_CACHE", "false"))
		embedStaticFiles, _       = strconv.ParseBool(env("EMBED_STATIC_FILES", "false"))
	)
	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Println("\nDon't forget to set TOKEN_KEY, SMTP_USERNAME and SMTP_PASSWORD for real usage.")
	}
	flag.IntVar(&port, "port", port, "Port in which this server will run")
	flag.StringVar(&originStr, "origin", originStr, "URL origin for this service")
	flag.StringVar(&dbURL, "db", dbURL, "Database URL")
	flag.StringVar(&natsURL, "nats", natsURL, "NATS URL")
	flag.StringVar(&smtpHost, "smtp-host", smtpHost, "SMTP server host")
	flag.IntVar(&smtpPort, "smtp-port", smtpPort, "SMTP server port")
	flag.BoolVar(&enableStaticFilesCache, "enable-static-files-cache", enableStaticFilesCache, "Enable static files cache")
	flag.BoolVar(&embedStaticFiles, "embed-static-files", embedStaticFiles, "Embed static files")
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

	if err = db.Ping(); err != nil {
		return fmt.Errorf("could not ping to db: %w", err)
	}

	natsConn, err := natslib.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("could not connect to NATS server: %w", err)
	}

	pubsub := &nats.PubSub{Conn: natsConn}

	var sender mailing.Sender
	if smtpUsername == "" || smtpPassword == "" {
		log.Println("could not setup smtp mailing: username and/or password not provided; using log implementation")
		sender = mailing.NewLogSender(
			"noreply@"+origin.Hostname(),
			&logWrapper{Logger: log.New(os.Stdout, "mailing ", log.LstdFlags)},
		)
	} else {
		sender = mailing.NewSMTPSender(
			"noreply@"+origin.Hostname(),
			smtpHost,
			strconv.Itoa(smtpPort),
			smtpUsername,
			smtpPassword,
		)

	}
	service := service.New(service.Conf{
		DB:       db,
		Sender:   sender,
		Origin:   origin,
		TokenKey: tokenKey,
		PubSub:   pubsub,
	})
	h := handler.New(service, enableStaticFilesCache, embedStaticFiles)
	server := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           h,
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       time.Second * 15,
	}

	errs := make(chan error, 2)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			errs <- fmt.Errorf("could not shutdown server: %w", err)
			return
		}

		errs <- ctx.Err()
	}()

	go func() {
		log.Printf("accepting connections on port %d\n", port)
		log.Printf("starting server at %s\n", origin)
		if err = server.ListenAndServe(); err != http.ErrServerClosed {
			errs <- fmt.Errorf("could not listen and serve: %w", err)
			return
		}

		errs <- nil
	}()

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
