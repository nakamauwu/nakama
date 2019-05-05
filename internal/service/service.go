package service

import (
	"database/sql"
	"net/url"

	"github.com/hako/branca"
	"github.com/matryer/vice"
	"github.com/nicolasparada/nakama/internal/mailing"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
type Service struct {
	db        *sql.DB
	transport vice.Transport
	sender    mailing.Sender
	origin    url.URL
	codec     *branca.Branca
}

// New service implementation.
func New(db *sql.DB, transport vice.Transport, sender mailing.Sender, origin url.URL, tokenKey string) *Service {
	cdc := branca.NewBranca(tokenKey)
	cdc.SetTTL(uint32(tokenLifespan.Seconds()))

	return &Service{
		db:        db,
		transport: transport,
		sender:    sender,
		origin:    origin,
		codec:     cdc,
	}
}
