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

	"github.com/hako/branca"
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
	// ErrInvalidRedirectURI denotes an invalid redirect uri.
	ErrInvalidRedirectURI = errors.New("invalid redirect uri")
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

	uri, err := url.ParseRequestURI(redirectURI)
	if err != nil {
		return ErrInvalidRedirectURI
	}

	var code string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO verification_codes (user_id) VALUES (
			(SELECT id FROM users WHERE email = $1)
		) RETURNING id`, email).Scan(&code)
	if isForeignKeyViolation(err) {
		return ErrUserNotFound
	}

	if err != nil {
		return fmt.Errorf("could not insert verification code: %w", err)
	}

	defer func() {
		if err != nil {
			_, err := s.db.Exec("DELETE FROM verification_codes WHERE id = $1", code)
			if err != nil {
				log.Printf("could not delete verification code: %v\n", err)
			}
		}
	}()

	link := cloneURL(s.origin)
	link.Path = "/api/auth_redirect"
	q := link.Query()
	q.Set("verification_code", code)
	q.Set("redirect_uri", uri.String())
	link.RawQuery = q.Encode()

	if magicLinkMailTmpl == nil {
		magicLinkMailTmpl, err = template.ParseFiles("web/template/mail/magic-link.html")
		if err != nil {
			return fmt.Errorf("could not parse magic link mail template: %w", err)
		}
	}

	var b bytes.Buffer
	if err = magicLinkMailTmpl.Execute(&b, map[string]interface{}{
		"MagicLink": link.String(),
		"Minutes":   int(verificationCodeLifespan.Minutes()),
	}); err != nil {
		return fmt.Errorf("could not execute magic link mail template: %w", err)
	}

	if err = s.sender.Send(email, "Magic Link", b.String()); err != nil {
		return fmt.Errorf("could not send magic link: %w", err)
	}

	return nil
}

// AuthURI to be redirected to and complete the login flow.
// It contains the token in the hash fragment.
func (s *Service) AuthURI(ctx context.Context, verificationCode, redirectURI string) (string, error) {
	verificationCode = strings.TrimSpace(verificationCode)
	if !reUUID.MatchString(verificationCode) {
		return "", ErrInvalidVerificationCode
	}

	uri, err := url.ParseRequestURI(redirectURI)
	if err != nil {
		return "", ErrInvalidRedirectURI
	}

	var uid string
	var createdAt time.Time
	err = s.db.QueryRowContext(ctx, `
		DELETE FROM verification_codes WHERE id = $1
		RETURNING user_id, created_at`, verificationCode).Scan(&uid, &createdAt)
	if err == sql.ErrNoRows {
		return "", ErrVerificationCodeNotFound
	}

	if err != nil {
		return "", fmt.Errorf("could not delete verification code: %w", err)
	}

	now := time.Now()
	exp := createdAt.Add(verificationCodeLifespan)
	if exp.Equal(now) || exp.Before(now) {
		return "", ErrExpiredToken
	}

	token, err := s.codec().EncodeToString(uid)
	if err != nil {
		return "", fmt.Errorf("could not create token: %w", err)
	}

	f := url.Values{}
	f.Set("token", token)
	f.Set("expires_at", now.Add(tokenLifespan).Format(time.RFC3339Nano))
	uri.Fragment = f.Encode()

	return uri.String(), nil
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
	err := s.db.QueryRowContext(ctx, query, email).Scan(&out.User.ID, &out.User.Username, &avatar)

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
		msg := err.Error()
		if msg == "invalid base62 token" || msg == "invalid token version" {
			return "", ErrInvalidToken
		}
		if msg == "token is expired" {
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

func (s *Service) deleteExpiredVerificationCodesJob() {
	ticker := time.NewTicker(time.Hour * 24)
	ctx := context.Background()
	done := ctx.Done()
	for {
		select {
		case <-ticker.C:
			if err := s.deleteExpiredVerificationCodes(ctx); err != nil {
				log.Println(err)
			}
		case <-done:
			ticker.Stop()
			return
		}
	}
}

func (s *Service) deleteExpiredVerificationCodes(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM verification_codes WHERE (created_at - INTERVAL '%dm') <= now()", int64(verificationCodeLifespan.Minutes()))
	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("could not delete expired verification code: %w", err)
	}
	return nil
}

func (s *Service) codec() *branca.Branca {
	cdc := branca.NewBranca(s.tokenKey)
	cdc.SetTTL(uint32(tokenLifespan.Seconds()))
	return cdc
}
