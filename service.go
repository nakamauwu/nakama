package nakama

import (
	"context"
	"database/sql"
	_ "embed"
	"net/url"

	"github.com/duo-labs/webauthn/webauthn"
	"github.com/nicolasparada/nakama/mailing"
	"github.com/nicolasparada/nakama/pubsub"
	"github.com/nicolasparada/nakama/storage"
)

//go:embed schema.sql
var Schema string

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
	WebAuthn        *webauthn.WebAuthn
}

// RunBackgroundJobs -
func (s *Service) RunBackgroundJobs(ctx context.Context) {
	go s.deleteExpiredVerificationCodesJob(ctx)
}
