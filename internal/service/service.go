package service

import (
	"database/sql"
	"net/url"

	"github.com/hako/branca"
	"github.com/nicolasparada/nakama/internal/mailing"
	"github.com/nicolasparada/nakama/internal/pubsub"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
type Service struct {
	db     *sql.DB
	pubsub pubsub.PubSub
	sender mailing.Sender
	origin url.URL
	codec  *branca.Branca
}

// New service implementation.
func New(db *sql.DB, pubsub pubsub.PubSub, sender mailing.Sender, origin url.URL, tokenKey string) *Service {
	cdc := branca.NewBranca(tokenKey)
	cdc.SetTTL(uint32(tokenLifespan.Seconds()))

	return &Service{
		db:     db,
		pubsub: pubsub,
		sender: sender,
		origin: origin,
		codec:  cdc,
	}
}
