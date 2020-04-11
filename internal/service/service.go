package service

import (
	"database/sql"
	"net/url"

	"github.com/nicolasparada/nakama/internal/mailing"
	"github.com/nicolasparada/nakama/internal/pubsub"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
type Service struct {
	db       *sql.DB
	sender   mailing.Sender
	origin   *url.URL
	tokenKey string
	pubsub   pubsub.PubSub
}

// Conf contains all service configuration.
type Conf struct {
	DB       *sql.DB
	Sender   mailing.Sender
	Origin   *url.URL
	TokenKey string
	PubSub   pubsub.PubSub
}

// New service implementation.
func New(conf Conf) *Service {
	s := &Service{
		db:       conf.DB,
		sender:   conf.Sender,
		origin:   conf.Origin,
		tokenKey: conf.TokenKey,
		pubsub:   conf.PubSub,
	}

	go s.deleteExpiredVerificationCodesJob()

	return s
}
