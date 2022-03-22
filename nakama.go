package nakama

import (
	"database/sql"
	_ "embed"
	"html/template"
	"net/url"
	"sync"

	"github.com/duo-labs/webauthn/webauthn"
	"github.com/go-kit/log"
	"github.com/nakamauwu/nakama/mailing"
	"github.com/nakamauwu/nakama/pubsub"
	"github.com/nakamauwu/nakama/storage"
)

//go:embed schema.sql
var Schema string

// Service contains the core business logic separated from the transport layer.
// You can use it to back a REST, gRPC or GraphQL API.
// You must call RunBackgroundJobs afterward.
type Service struct {
	Logger           log.Logger
	DB               *sql.DB
	Sender           mailing.Sender
	Origin           *url.URL
	TokenKey         string
	PubSub           pubsub.PubSub
	Store            storage.Store
	AvatarURLPrefix  string
	CoverURLPrefix   string
	MediaURLPrefix   string
	WebAuthn         *webauthn.WebAuthn
	DisabledDevLogin bool
	AllowedOrigins   []string
	VAPIDPrivateKey  string
	VAPIDPublicKey   string

	magicLinkTmplOncer sync.Once
	magicLinkTmpl      *template.Template
}
