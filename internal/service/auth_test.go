package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/nicolasparada/nakama/internal/mailing"
	"github.com/nicolasparada/nakama/internal/testutil"
)

func TestService_SendMagicLink(t *testing.T) {
	t.Run("empty_email", func(t *testing.T) {
		svc := &Service{}
		ctx := context.Background()
		err := svc.SendMagicLink(ctx, "", "")
		testutil.AssertEqual(t, ErrInvalidEmail, err, "error")
	})

	t.Run("invalid_email", func(t *testing.T) {
		svc := &Service{}
		ctx := context.Background()
		err := svc.SendMagicLink(ctx, "nope", "")
		testutil.AssertEqual(t, ErrInvalidEmail, err, "error")
	})

	t.Run("empty_redirect_uri", func(t *testing.T) {
		svc := &Service{}
		ctx := context.Background()
		email := testutil.RandStr(t, 10) + "@example.org"
		err := svc.SendMagicLink(ctx, email, "")
		testutil.AssertEqual(t, ErrInvalidRedirectURI, err, "error")
	})

	t.Run("non_absolute_redirect_uri", func(t *testing.T) {
		svc := &Service{}
		ctx := context.Background()
		email := testutil.RandStr(t, 10) + "@example.org"
		err := svc.SendMagicLink(ctx, email, "/nope")
		testutil.AssertEqual(t, ErrInvalidRedirectURI, err, "error")
	})

	redirectURI := "https://example.org/login-callback"
	t.Run("user_not_found", func(t *testing.T) {
		svc := &Service{
			db: testDB,
		}
		ctx := context.Background()
		email := testutil.RandStr(t, 10) + "@example.org"
		err := svc.SendMagicLink(ctx, email, redirectURI)
		testutil.AssertEqual(t, ErrUserNotFound, err, "error")
	})

	origin := &url.URL{
		Scheme: "http",
		Host:   "localhost:3000",
	}

	t.Run("sender_send_error", func(t *testing.T) {
		errInternal := errors.New("internal error")
		svc := &Service{
			db:     testDB,
			origin: origin,
			sender: &mailing.SenderMock{
				SendFunc: func(to, subject, body string) error {
					return errInternal
				},
			},
		}

		username := testutil.RandStr(t, 8)
		email := username + "@example.org"
		_, err := testDB.Exec(`INSERT INTO users (email, username) VALUES ($1, $2)`, email, username)
		testutil.AssertEqual(t, nil, err, "sql insert")

		ctx := context.Background()
		err = svc.SendMagicLink(ctx, email, redirectURI)
		testutil.AssertEqual(t, fmt.Errorf("could not send magic link: %w", errInternal), err, "error")
	})

	t.Run("ok", func(t *testing.T) {
		senderMock := &mailing.SenderMock{
			SendFunc: func(to, subject, body string) error {
				return nil
			},
		}
		svc := &Service{
			db:     testDB,
			origin: origin,
			sender: senderMock,
		}

		username := testutil.RandStr(t, 8)
		email := username + "@example.org"
		_, err := testDB.Exec(`INSERT INTO users (email, username) VALUES ($1, $2)`, email, username)
		testutil.AssertEqual(t, nil, err, "sql insert")

		ctx := context.Background()
		err = svc.SendMagicLink(ctx, email, redirectURI)
		testutil.AssertEqual(t, nil, err, "error")

		calls := senderMock.SendCalls()
		testutil.AssertEqual(t, 1, len(calls), "calls length")

		call := calls[0]
		testutil.AssertEqual(t, email, call.To, "sender send-to")
		testutil.AssertEqual(t, "Magic Link", call.Subject, "sender send-subject")
		testutil.AssertEqual(t, "text/html; charset=utf-8", http.DetectContentType([]byte(call.Body)), "sender send-subject content type")
		t.Logf("\nmagic link body:\n%s\n\n", call.Body)
	})
}
