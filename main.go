package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nicolasparada/nakama/internal/handler"
	"github.com/nicolasparada/nakama/internal/service"
)

func main() {
	godotenv.Load()

	var (
		port         = intEnv("PORT", 3000)
		origin       = env("ORIGIN", fmt.Sprintf("http://localhost:%d", port))
		databaseURL  = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable")
		secretKey    = env("SECRET_KEY", "supersecretkeyyoushouldnotcommit")
		smtpHost     = env("SMTP_HOST", "smtp.mailtrap.io")
		smtpPort     = intEnv("SMTP_PORT", 25)
		smtpUsername = mustEnv("SMTP_USERNAME")
		smtpPassword = mustEnv("SMTP_PASSWORD")
	)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("could not open db connection: %v\n", err)
		return
	}

	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatalf("could not ping to db: %v\n", err)
		return
	}

	srvc, err := service.New(service.Config{
		DB:           db,
		SecretKey:    secretKey,
		Origin:       origin,
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUsername: smtpUsername,
		SMTPPassword: smtpPassword,
	})
	if err != nil {
		log.Fatalf("could not create service: %v\n", err)
		return
	}

	svr := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler.New(srvc, time.Second*15),
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       time.Second * 15,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 30,
	}

	log.Printf("accepting connections on port %d\n", port)
	log.Printf("server running at %s", origin)
	if err = svr.ListenAndServe(); err != nil {
		log.Fatalf("could not start server: %v\n", err)
	}
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
		log.Fatalf("%s missing on environment variables\n", key)
		return ""
	}
	return s
}

func intEnv(key string, fallbackValue int) int {
	s, ok := os.LookupEnv(key)
	if !ok {
		return fallbackValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return fallbackValue
	}
	return i
}
