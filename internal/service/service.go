package service

import (
	"database/sql"
	"net/url"
	"sync"

	"github.com/nicolasparada/nakama/internal/mailing"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
type Service struct {
	db                  *sql.DB
	sender              mailing.Sender
	origin              url.URL
	tokenKey            string
	timelineItemClients sync.Map
	commentClients      sync.Map
	notificationClients sync.Map
}

// New service implementation.
func New(db *sql.DB, sender mailing.Sender, origin url.URL, tokenKey string) *Service {
	s := &Service{
		db:       db,
		sender:   sender,
		origin:   origin,
		tokenKey: tokenKey,
	}

	go s.deleteExpiredVerificationCodesJob()

	return s
}
