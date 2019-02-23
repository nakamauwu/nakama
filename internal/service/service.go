package service

import (
	"database/sql"
	"sync"

	"github.com/hako/branca"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
type Service struct {
	db                  *sql.DB
	cdc                 *branca.Branca
	origin              string
	timelineItemClients sync.Map
	commentClients      sync.Map
	notificationClients sync.Map
}

// New service implementation.
func New(db *sql.DB, cdc *branca.Branca, origin string) *Service {
	return &Service{
		db:     db,
		cdc:    cdc,
		origin: origin,
	}
}
