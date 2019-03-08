package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nicolasparada/nakama/internal/handler"
	"github.com/nicolasparada/nakama/internal/mailing"
	"github.com/nicolasparada/nakama/internal/service"
)

func main() {
	godotenv.Load()
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var (
		port         = env("PORT", "3000")
		originStr    = env("ORIGIN", "http://localhost:"+port)
		dbURL        = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable")
		tokenKey     = env("TOKEN_KEY", "supersecretkeyyoushouldnotcommit")
		smtpHost     = env("SMTP_HOST", "smtp.mailtrap.io")
		smtpPort     = env("SMTP_PORT", "25")
		smtpUsername = mustEnv("SMTP_USERNAME")
		smtpPassword = mustEnv("SMTP_PASSWORD")
	)

	origin, err := url.Parse(originStr)
	if err != nil || !origin.IsAbs() {
		return errors.New("invalid origin url")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("could not open db connection: %v", err)
	}

	defer db.Close()

	if err = db.Ping(); err != nil {
		return fmt.Errorf("could not ping to db: %v", err)
	}

	sender := mailing.NewSender(
		"noreply@"+origin.Hostname(),
		smtpHost,
		smtpPort,
		smtpUsername,
		smtpPassword,
	)
	service := service.New(
		db,
		sender,
		*origin,
		tokenKey,
	)
	server := http.Server{
		Addr:              ":" + port,
		Handler:           handler.New(service, time.Second*15),
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       time.Second * 15,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 30,
	}

	errs := make(chan error, 2)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, os.Kill)

		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			errs <- fmt.Errorf("could not shutdown server: %v", err)
			return
		}

		errs <- ctx.Err()
	}()

	go func() {
		log.Printf("accepting connections on port %s\n", port)
		log.Printf("starting server at %s\n", origin)
		if err = server.ListenAndServe(); err != http.ErrServerClosed {
			errs <- fmt.Errorf("could not listen and serve: %v", err)
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

func mustEnv(key string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("%s missing on environment variables", key))
	}
	return s
}
