package service

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/smtp"
	"net/url"
	"strconv"
	"sync"

	"github.com/hako/branca"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
type Service struct {
	db                  *sql.DB
	codec               *branca.Branca
	origin              url.URL
	noReply             string
	smtpAddr            string
	smtpAuth            smtp.Auth
	timelineItemClients sync.Map
	commentClients      sync.Map
	notificationClients sync.Map
}

// Config to create a new service.
type Config struct {
	DB           *sql.DB
	SecretKey    string
	Origin       string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
}

// New service implementation.
func New(cfg Config) (*Service, error) {
	cdc := branca.NewBranca(cfg.SecretKey)
	cdc.SetTTL(uint32(tokenLifespan.Seconds()))

	origin, err := url.Parse(cfg.Origin)
	if err != nil || !origin.IsAbs() {
		return nil, errors.New("origin must by an absolute url")
	}

	s := &Service{
		db:       cfg.DB,
		codec:    cdc,
		origin:   *origin,
		noReply:  "noreply@+" + origin.Hostname(),
		smtpAddr: net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort)),
		smtpAuth: smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost),
	}

	go s.deleteExpiredVerificationCodesCronJob(context.Background())

	return s, nil
}
