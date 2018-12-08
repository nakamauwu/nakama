package service

import (
	"database/sql"

	"github.com/hako/branca"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, GRPC or GraphQL API.
type Service struct {
	db     *sql.DB
	cdc    *branca.Branca
	origin string
}

// New service implementation.
func New(db *sql.DB, cdc *branca.Branca, origin string) *Service {
	return &Service{
		db:     db,
		cdc:    cdc,
		origin: origin,
	}
}
