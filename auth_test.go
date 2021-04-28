package nakama

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/nicolasparada/nakama/mailing"
	"github.com/nicolasparada/nakama/testutil"
)

func TestService_SendMagicLink(t *testing.T) {
	ctx := context.Background()
	t.Run("empty_email", func(t *testing.T) {
		svc := &Service{}
		err := svc.SendMagicLink(ctx, "", "")
		testutil.AssertEqual(t, ErrInvalidEmail, err, "error")
	})

	t.Run("invalid_email", func(t *testing.T) {
		svc := &Service{}
		err := svc.SendMagicLink(ctx, "nope", "")
		testutil.AssertEqual(t, ErrInvalidEmail, err, "error")
	})

	email := testutil.RandStr(t, 10) + "@example.org"

	t.Run("empty_redirect_uri", func(t *testing.T) {
		svc := &Service{}
		err := svc.SendMagicLink(ctx, email, "")
		testutil.AssertEqual(t, ErrInvalidRedirectURI, err, "error")
	})

	t.Run("non_absolute_redirect_uri", func(t *testing.T) {
		svc := &Service{}
		err := svc.SendMagicLink(ctx, email, "/nope")
		testutil.AssertEqual(t, ErrInvalidRedirectURI, err, "error")
	})

	t.Run("untrusted_redirect_uri", func(t *testing.T) {
		svc := &Service{
			Origin: &url.URL{
				Scheme: "http",
				Host:   "localhost:3000",
			},
		}
		err := svc.SendMagicLink(ctx, email, "https://example.org/login-callback")
		testutil.AssertEqual(t, ErrUntrustedRedirectURI, err, "error")
	})

	redirectURI := "http://localhost:3000/login-callback"
	origin := &url.URL{
		Scheme: "http",
		Host:   "localhost:3000",
	}

	t.Run("sender_send_error", func(t *testing.T) {
		errInternal := errors.New("internal error")
		svc := &Service{
			DB:     testDB,
			Origin: origin,
			Sender: &mailing.SenderMock{
				SendFunc: func(to, subject, html, text string) error {
					return errInternal
				},
			},
		}

		err := svc.SendMagicLink(ctx, email, redirectURI)
		testutil.AssertEqual(t, fmt.Errorf("could not send magic link: %w", errInternal), err, "error")
	})

	t.Run("ok", func(t *testing.T) {
		senderMock := &mailing.SenderMock{
			SendFunc: func(to, subject, html, text string) error {
				return nil
			},
		}
		svc := &Service{
			DB:     testDB,
			Origin: origin,
			Sender: senderMock,
		}

		err := svc.SendMagicLink(ctx, email, redirectURI)
		testutil.AssertEqual(t, nil, err, "error")

		calls := senderMock.SendCalls()
		testutil.AssertEqual(t, 1, len(calls), "calls length")

		call := calls[0]
		testutil.AssertEqual(t, email, call.To, "sender send-to")
		testutil.AssertEqual(t, "Magic Link", call.Subject, "sender send-subject")
		testutil.AssertEqual(t, "text/html; charset=utf-8", http.DetectContentType([]byte(call.HTML)), "sender send-subject content type")
		t.Logf("\nmagic link text:\n%s\n\n", call.Text)
	})
}
