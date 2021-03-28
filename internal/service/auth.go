package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/hako/branca"
	webtemplate "github.com/nicolasparada/nakama/web/template"
)

// KeyAuthUserID to use in context.
const KeyAuthUserID = ctxkey("auth_user_id")

const (
	verificationCodeLifespan = time.Minute * 15
	tokenLifespan            = time.Hour * 24 * 14
)

var (
	// ErrUnimplemented denotes a not implemented functionality.
	ErrUnimplemented = errors.New("unimplemented")
	// ErrUnauthenticated denotes no authenticated user in context.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrInvalidRedirectURI denotes an invalid redirect URI.
	ErrInvalidRedirectURI = errors.New("invalid redirect URI")
	// ErrInvalidToken denotes an invalid token.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken denotes that the token already expired.
	ErrExpiredToken = errors.New("expired token")
	// ErrInvalidVerificationCode denotes an invalid verification code.
	ErrInvalidVerificationCode = errors.New("invalid verification code")
	// ErrVerificationCodeNotFound denotes a not found verification code.
	ErrVerificationCodeNotFound = errors.New("verification code not found")
)

var magicLinkMailTmpl *template.Template

type ctxkey string

// TokenOutput response.
type TokenOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// DevLoginOutput response.
type DevLoginOutput struct {
	User      User      `json:"user"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SendMagicLink to login without passwords.
func (s *Service) SendMagicLink(ctx context.Context, email, redirectURI string) error {
	email = strings.TrimSpace(email)
	if !reEmail.MatchString(email) {
		return ErrInvalidEmail
	}

	uri, err := url.Parse(redirectURI)
	if err != nil || !uri.IsAbs() {
		return ErrInvalidRedirectURI
	}

	var code string
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO verification_codes (email) VALUES ($1) RETURNING id
	`, email).Scan(&code)
	if err != nil {
		return fmt.Errorf("could not insert verification code: %w", err)
	}

	defer func() {
		if err != nil {
			go func() {
				_, err := s.DB.Exec("DELETE FROM verification_codes WHERE id = $1", code)
				if err != nil {
					log.Printf("could not delete verification code: %v\n", err)
				}
			}()
		}
	}()

	magicLink := cloneURL(s.Origin)
	magicLink.Path = "/api/auth_redirect"
	q := magicLink.Query()
	q.Set("verification_code", code)
	q.Set("redirect_uri", uri.String())
	magicLink.RawQuery = q.Encode()

	if magicLinkMailTmpl == nil {
		magicLinkMailTmpl, err = template.ParseFS(webtemplate.Files, "mail/magic-link.html")
		if err != nil {
			return fmt.Errorf("could not parse magic link mail template: %w", err)
		}
	}

	var b bytes.Buffer
	err = magicLinkMailTmpl.Execute(&b, map[string]interface{}{
		"MagicLink": magicLink,
		"Minutes":   int(verificationCodeLifespan.Minutes()),
	})
	if err != nil {
		return fmt.Errorf("could not execute magic link mail template: %w", err)
	}

	err = s.Sender.Send(email, "Magic Link", b.String(), magicLink.String())
	if err != nil {
		return fmt.Errorf("could not send magic link: %w", err)
	}

	return nil
}

// AuthURI to be redirected to and complete the login flow.
// It contains the token and expires_at in the hash fragment.
func (s *Service) AuthURI(ctx context.Context, reqURIStr string) (*url.URL, error) {
	reqURI, err := url.Parse(reqURIStr)
	if err != nil {
		return nil, fmt.Errorf("could not url parse request URI: %w", err)
	}

	reqQuery := reqURI.Query()
	redirectURI, err := url.Parse(strings.TrimSpace(reqQuery.Get("redirect_uri")))
	if err != nil || !redirectURI.IsAbs() {
		return nil, ErrInvalidRedirectURI
	}

	verificationCode := strings.TrimSpace(reqQuery.Get("verification_code"))
	if !reUUID.MatchString(verificationCode) {
		return uriWithQuery(redirectURI, map[string]string{
			"error": ErrInvalidVerificationCode.Error(),
		})
	}

	username := strings.TrimSpace(reqQuery.Get("username"))
	if username != "" {
		if !reUsername.MatchString(username) {
			return uriWithQuery(redirectURI, map[string]string{
				"error": ErrInvalidUsername.Error(),
			})
		}
	}

	var uid string
	err = crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		var email string
		var createdAt time.Time
		err := tx.QueryRowContext(ctx, `SELECT email, created_at FROM verification_codes WHERE id = $1`, verificationCode).
			Scan(&email, &createdAt)
		if err == sql.ErrNoRows {
			return ErrVerificationCodeNotFound
		}

		if err != nil {
			return fmt.Errorf("could not sql query select verification code: %w", err)
		}

		if isVerificationCodeExpired(createdAt) {
			return ErrExpiredToken
		}

		err = tx.QueryRowContext(ctx, `SELECT id AS user_id FROM users WHERE email = $1`, email).
			Scan(&uid)
		if err == sql.ErrNoRows {
			if username == "" {
				return ErrUserNotFound
			}

			err := tx.QueryRowContext(ctx, `INSERT INTO users (email, username) VALUES ($1, $2) RETURNING id`, email, username).
				Scan(&uid)
			if isUniqueViolation(err) {
				return ErrUsernameTaken
			}

			if err != nil {
				return fmt.Errorf("could not sql insert user at magic link: %w", err)
			}

			return nil
		}

		if err != nil {
			return fmt.Errorf("could not sql query select user from verification code email: %w", err)
		}

		return nil
	})
	if err == ErrUserNotFound || err == ErrUsernameTaken {
		return uriWithQuery(redirectURI, map[string]string{
			"error":          err.Error(),
			"retry_endpoint": reqURIStr,
		})
	}

	if err == ErrVerificationCodeNotFound {
		return uriWithQuery(redirectURI, map[string]string{
			"error": ErrVerificationCodeNotFound.Error(),
		})
	}

	go func() {
		_, err := s.DB.Exec("DELETE FROM verification_codes WHERE id = $1", verificationCode)
		if err != nil {
			log.Printf("could not delete verification code: %v\n", err)
			return
		}
	}()

	if err == ErrExpiredToken {
		return uriWithQuery(redirectURI, map[string]string{
			"error": ErrExpiredToken.Error(),
		})
	}

	if err != nil {
		log.Println(err)
		return uriWithQuery(redirectURI, map[string]string{
			"error": "something went wrong",
		})
	}

	now := time.Now()
	token, err := s.codec().EncodeToString(uid)
	if err != nil {
		log.Printf("could not create token: %v\n", err)
		return uriWithQuery(redirectURI, map[string]string{
			"error": "something went wrong",
		})
	}

	return uriWithQuery(redirectURI, map[string]string{
		"token":      token,
		"expires_at": now.Add(tokenLifespan).Format(time.RFC3339Nano),
	})
}

func isVerificationCodeExpired(t time.Time) bool {
	now := time.Now()
	exp := t.Add(verificationCodeLifespan)
	return exp.Equal(now) || exp.Before(now)
}

// DevLogin is a login for development purposes only.
// TODO: disable dev login on production.
func (s *Service) DevLogin(ctx context.Context, email string) (DevLoginOutput, error) {
	var out DevLoginOutput

	email = strings.TrimSpace(email)
	if !reEmail.MatchString(email) {
		return out, ErrInvalidEmail
	}

	var avatar sql.NullString
	query := "SELECT id, username, avatar FROM users WHERE email = $1"
	err := s.DB.QueryRowContext(ctx, query, email).Scan(&out.User.ID, &out.User.Username, &avatar)

	if err == sql.ErrNoRows {
		return out, ErrUserNotFound
	}

	if err != nil {
		return out, fmt.Errorf("could not query select user: %w", err)
	}

	out.User.AvatarURL = s.avatarURL(avatar)

	out.Token, err = s.codec().EncodeToString(out.User.ID)
	if err != nil {
		return out, fmt.Errorf("could not create token: %w", err)
	}

	out.ExpiresAt = time.Now().Add(tokenLifespan)

	return out, nil
}

// AuthUserIDFromToken decodes the token into a user ID.
func (s *Service) AuthUserIDFromToken(token string) (string, error) {
	uid, err := s.codec().DecodeToString(token)
	if err != nil {
		// We check error string because branca doesn't export errors.
		if errors.Is(err, branca.ErrInvalidToken) || errors.Is(err, branca.ErrInvalidTokenVersion) {
			return "", ErrInvalidToken
		}
		if _, ok := err.(*branca.ErrExpiredToken); ok {
			return "", ErrExpiredToken
		}
		return "", fmt.Errorf("could not decode token: %w", err)
	}

	if !reUUID.MatchString(uid) {
		return "", ErrInvalidUserID
	}

	return uid, nil
}

// AuthUser is the current authenticated user.
func (s *Service) AuthUser(ctx context.Context) (User, error) {
	var u User
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return u, ErrUnauthenticated
	}

	return s.userByID(ctx, uid)
}

// Token to authenticate requests.
func (s *Service) Token(ctx context.Context) (TokenOutput, error) {
	var out TokenOutput
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return out, ErrUnauthenticated
	}

	var err error
	out.Token, err = s.codec().EncodeToString(uid)
	if err != nil {
		return out, fmt.Errorf("could not create token: %w", err)
	}

	out.ExpiresAt = time.Now().Add(tokenLifespan)

	return out, nil
}

func (s *Service) deleteExpiredVerificationCodesJob(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)
	done := ctx.Done()

loop:
	for {
		select {
		case <-ticker.C:
			if err := s.deleteExpiredVerificationCodes(ctx); err != nil {
				log.Println(err)
			}
		case <-done:
			ticker.Stop()
			break loop
		}
	}
}

func (s *Service) deleteExpiredVerificationCodes(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM verification_codes WHERE (created_at - INTERVAL '%dm') <= now()", int64(verificationCodeLifespan.Minutes()))
	if _, err := s.DB.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("could not delete expired verification code: %w", err)
	}
	return nil
}

func (s *Service) codec() *branca.Branca {
	cdc := branca.NewBranca(s.TokenKey)
	cdc.SetTTL(uint32(tokenLifespan.Seconds()))
	return cdc
}
