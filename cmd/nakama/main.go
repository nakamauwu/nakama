package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/go-kit/kit/log"
	"github.com/gorilla/securecookie"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/nicolasparada/nakama"
	"github.com/nicolasparada/nakama/mailing"
	natspubsub "github.com/nicolasparada/nakama/pubsub/nats"
	"github.com/nicolasparada/nakama/storage"
	fsstorage "github.com/nicolasparada/nakama/storage/fs"
	s3storage "github.com/nicolasparada/nakama/storage/s3"
	httptransport "github.com/nicolasparada/nakama/transport/http"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	if err := run(ctx, logger, os.Args[1:]); err != nil {
		_ = logger.Log("error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger log.Logger, args []string) error {
	var (
		port, _                   = strconv.Atoi(env("PORT", "3000"))
		originStr                 = env("ORIGIN", fmt.Sprintf("http://localhost:%d", port))
		dbURL                     = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable")
		execSchema, _             = strconv.ParseBool(env("EXEC_SCHEMA", "false"))
		tokenKey                  = env("TOKEN_KEY", "supersecretkeyyoushouldnotcommit")
		natsURL                   = env("NATS_URL", nats.DefaultURL)
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
		avatarURLPrefix           = env("AVATAR_URL_PREFIX", originStr+"/img/avatars/")
		cookieHashKey             = env("COOKIE_HASH_KEY", "supersecretkeyyoushouldnotcommit")
		cookieBlockKey            = env("COOKIE_BLOCK_KEY", "supersecretkeyyoushouldnotcommit")
		disabledDevLogin, _       = strconv.ParseBool(os.Getenv("DISABLE_DEV_LOGIN"))
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
	flag.StringVar(&avatarURLPrefix, "avatar-url-prefix", avatarURLPrefix, "Avatar URL prefix")
	flag.StringVar(&cookieHashKey, "cookie-hash-key", cookieHashKey, "Cookie hash key. 32 or 64 bytes")
	flag.StringVar(&cookieBlockKey, "cookie-block-key", cookieBlockKey, "Cookie block key. 16, 24, or 32 bytes")
	flag.BoolVar(&disabledDevLogin, "disable-dev-login", disabledDevLogin, "Disable development login endpoint")
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

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("could not ping to db: %w", err)
	}

	if execSchema {
		_, err := db.ExecContext(ctx, nakama.Schema)
		if err != nil {
			return fmt.Errorf("could not run schema: %w", err)
		}
	}

	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("could not connect to NATS server: %w", err)
	}

	pubsub := &natspubsub.PubSub{Conn: natsConn}

	var sender mailing.Sender
	sendFrom := "no-reply@" + origin.Hostname()
	if sendgridAPIKey != "" {
		_ = logger.Log("mailing_implementation", "sendgrid")
		sender = mailing.NewSendgridSender(sendFrom, sendgridAPIKey)
	} else if smtpUsername != "" && smtpPassword != "" {
		_ = logger.Log("mailing_implementation", "smtp")
		sender = mailing.NewSMTPSender(
			sendFrom,
			smtpHost,
			smtpPort,
			smtpUsername,
			smtpPassword,
		)
	} else {
		_ = logger.Log("mailing_implementation", "log")
		sender = mailing.NewLogSender(
			sendFrom,
			log.With(logger, "component", "mailing"),
		)
	}

	var store storage.Store
	s3Enabled := s3Endpoint != "" && s3AccessKey != "" && s3SecretKey != ""
	if s3Enabled {
		_ = logger.Log("storage_implementation", "s3")
		store = &s3storage.Store{
			Endpoint:  s3Endpoint,
			Region:    s3Region,
			Bucket:    s3Bucket,
			AccessKey: s3AccessKey,
			SecretKey: s3SecretKey,
		}
	} else {
		_ = logger.Log("storage_implementation", "os file system")
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current working directory: %w", err)
		}

		store = &fsstorage.Store{Root: filepath.Join(wd, "web", "static", "img", "avatars")}
	}

	webauthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName:         "nakama",
		RPID:                  origin.Hostname(),
		RPOrigin:              origin.String(),
		RPIcon:                "",
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			RequireResidentKey:      nil,
			UserVerification:        protocol.VerificationRequired,
		},
		Timeout: int(httptransport.WebAuthnTimeout.Milliseconds()),
		Debug:   false,
	})
	if err != nil {
		return fmt.Errorf("could not create webauth config: %w", err)
	}

	svc := &nakama.Service{
		Logger:           logger,
		DB:               db,
		Sender:           sender,
		Origin:           origin,
		TokenKey:         tokenKey,
		PubSub:           pubsub,
		Store:            store,
		AvatarURLPrefix:  avatarURLPrefix,
		WebAuthn:         webauthn,
		DisabledDevLogin: disabledDevLogin,
	}

	go svc.RunBackgroundJobs(ctx)

	serveAvatars := !s3Enabled
	cookieCodec := securecookie.New(
		[]byte(cookieHashKey),
		[]byte(cookieBlockKey),
	)
	h := httptransport.New(svc, log.With(logger, "component", "http"), store, cookieCodec, enableStaticFilesCache, embedStaticFiles, serveAvatars)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           h,
		ReadHeaderTimeout: time.Second * 10,
		ReadTimeout:       time.Second * 30,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	errs := make(chan error, 1)
	go func() {
		<-ctx.Done()
		fmt.Println()

		_ = logger.Log("message", "gracefully shutting down")
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			errs <- fmt.Errorf("could not shutdown server: %w", err)
		}

		errs <- nil
	}()

	_ = logger.Log("message", "accepting connections", "port", port)
	_ = logger.Log("message", "starting server", "origin", origin)
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
