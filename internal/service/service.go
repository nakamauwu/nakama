package service

import (
	"context"
	"database/sql"
	"net/url"

	"github.com/nicolasparada/nakama/internal/mailing"
	"github.com/nicolasparada/nakama/internal/pubsub"
	"github.com/nicolasparada/nakama/internal/storage"
)

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
// You must call RunBackgroundJobs afterward.
type Service struct {
	DB              *sql.DB
	Sender          mailing.Sender
	Origin          *url.URL
	TokenKey        string
	PubSub          pubsub.PubSub
	Store           storage.Store
	AvatarURLPrefix string
}

// RunBackgroundJobs -
func (s *Service) RunBackgroundJobs(ctx context.Context) {
	go s.deleteExpiredVerificationCodesJob(ctx)
}
