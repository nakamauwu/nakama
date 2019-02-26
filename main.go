package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
		dburn        = env("DB_URN", "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable")
		secretKey    = env("SECRET_KEY", "supersecretkeyyoushouldnotcommit")
		smtpHost     = env("SMTP_HOST", "smtp.mailtrap.io")
		smtpPort     = intEnv("SMTP_PORT", 25)
		smtpUsername = mustEnv("SMTP_USERNAME")
		smtpPassword = mustEnv("SMTP_PASSWORD")
	)

	flag.IntVar(&port, "port", port, "Port ($PORT)")
	flag.StringVar(&origin, "origin", origin, "Origin URL ($ORIGIN)")
	flag.StringVar(&dburn, "db", dburn, "Database URN ($DB_URN)")
	flag.StringVar(&secretKey, "key", secretKey, "32 bytes long secret key to sign tokens ($SECRET_KEY)")
	flag.StringVar(&smtpHost, "smtp.host", smtpHost, "SMTP host ($SMTP_HOST)")
	flag.IntVar(&smtpPort, "smtp.port", smtpPort, "SMTP port ($SMTP_PORT)")
	flag.StringVar(&smtpUsername, "smtp.username", "", "SMTP username ($SMTP_USERNAME)")
	flag.StringVar(&smtpPassword, "smtp.password", "", "SMTP password ($SMTP_PASSWORD)")
	flag.Parse()

	db, err := sql.Open("postgres", dburn)
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

	errs := make(chan error, 2)

	go func() {
		log.Printf("accepting connections on port %d\n", port)
		log.Printf("server running at %s", origin)
		errs <- svr.ListenAndServe()
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill)
		errs <- fmt.Errorf("%v", <-c)
	}()

	log.Fatalf("terminated: %v\n", <-errs)
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
