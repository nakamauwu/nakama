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
	"regexp"
	"strconv"
	"strings"
	"time"
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
	// ErrInvalidVerificationCode denotes an invalid verification code.
	ErrInvalidVerificationCode = errors.New("invalid verification code")
	// ErrVerificationCodeNotFound denotes a not found verification code.
	ErrVerificationCodeNotFound = errors.New("verification code not found")
)

var rxUUID = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

var magicLinkMailTmpl *template.Template

type ctxkey string

// TokenOutput response.
type TokenOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// DevLoginOutput response.
type DevLoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	AuthUser  User      `json:"authUser"`
}

// SendMagicLink to login without passwords.
func (s *Service) SendMagicLink(ctx context.Context, email, redirectURI string) error {
	email = strings.TrimSpace(email)
	if !rxEmail.MatchString(email) {
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
		return fmt.Errorf("could not insert verification code: %v", err)
	}

	link := s.origin
	link.Path = "/api/auth_redirect"
	q := link.Query()
	q.Set("verification_code", code)
	q.Set("redirect_uri", uri.String())
	link.RawQuery = q.Encode()

	if magicLinkMailTmpl == nil {
		magicLinkMailTmpl, err = template.ParseFiles("web/template/mail/magic-link.html")
		if err != nil {
			return fmt.Errorf("could not parse magic link mail template: %v", err)
		}
	}

	var b bytes.Buffer
	if err = magicLinkMailTmpl.Execute(&b, map[string]interface{}{
		"MagicLink": link.String(),
		"Minutes":   int(verificationCodeLifespan.Minutes()),
	}); err != nil {
		return fmt.Errorf("could not execute magic link mail template: %v", err)
	}

	if err = s.sender.Send(email, "Magic Link", b.String()); err != nil {
		return fmt.Errorf("could not send magic link: %v", err)
	}

	go s.deleteVerificationCodeWhenExpires(code)

	return nil
}

// AuthURI to be redirected to and complete the login flow.
// It contains the token in the hash fragment.
func (s *Service) AuthURI(ctx context.Context, verificationCode, redirectURI string) (string, error) {
	verificationCode = strings.TrimSpace(verificationCode)
	if !rxUUID.MatchString(verificationCode) {
		return "", ErrInvalidVerificationCode
	}

	uri, err := url.ParseRequestURI(redirectURI)
	if err != nil {
		return "", ErrInvalidRedirectURI
	}

	var uid int64
	err = s.db.QueryRowContext(ctx, `
		DELETE FROM verification_codes WHERE id = $1
		RETURNING user_id`, verificationCode).Scan(&uid)
	if err == sql.ErrNoRows {
		return "", ErrVerificationCodeNotFound
	}

	if err != nil {
		return "", fmt.Errorf("could not delete verification code: %v", err)
	}

	token, err := s.codec.EncodeToString(strconv.FormatInt(uid, 10))
	if err != nil {
		return "", fmt.Errorf("could not create token: %v", err)
	}

	exp, err := time.Now().Add(tokenLifespan).MarshalText()
	if err != nil {
		return "", fmt.Errorf("could not marshall token expiration timestamp: %v", err)
	}

	f := url.Values{}
	f.Set("token", token)
	f.Set("expires_at", string(exp))
	uri.Fragment = f.Encode()

	return uri.String(), nil
}

// DevLogin is a login for development purposes only.
func (s *Service) DevLogin(ctx context.Context, email string) (DevLoginOutput, error) {
	var out DevLoginOutput

	if s.origin.Hostname() != "localhost" {
		return out, ErrUnimplemented
	}

	email = strings.TrimSpace(email)
	if !rxEmail.MatchString(email) {
		return out, ErrInvalidEmail
	}

	var avatar sql.NullString
	query := "SELECT id, username, avatar FROM users WHERE email = $1"
	err := s.db.QueryRowContext(ctx, query, email).Scan(&out.AuthUser.ID, &out.AuthUser.Username, &avatar)

	if err == sql.ErrNoRows {
		return out, ErrUserNotFound
	}

	if err != nil {
		return out, fmt.Errorf("could not query select user: %v", err)
	}

	out.AuthUser.AvatarURL = s.avatarURL(avatar)

	out.Token, err = s.codec.EncodeToString(strconv.FormatInt(out.AuthUser.ID, 10))
	if err != nil {
		return out, fmt.Errorf("could not create token: %v", err)
	}

	out.ExpiresAt = time.Now().Add(tokenLifespan)

	return out, nil
}

// AuthUserIDFromToken decodes the token into a user ID.
func (s *Service) AuthUserIDFromToken(token string) (int64, error) {
	str, err := s.codec.DecodeToString(token)
	if err != nil {
		return 0, fmt.Errorf("could not decode token: %v", err)
	}

	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse auth user id from token: %v", err)
	}

	return i, nil
}

// AuthUser is the current authenticated user.
func (s *Service) AuthUser(ctx context.Context) (User, error) {
	var u User
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return u, ErrUnauthenticated
	}

	return s.userByID(ctx, uid)
}

// Token to authenticate requests.
func (s *Service) Token(ctx context.Context) (TokenOutput, error) {
	var out TokenOutput
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return out, ErrUnauthenticated
	}

	var err error
	out.Token, err = s.codec.EncodeToString(strconv.FormatInt(uid, 10))
	if err != nil {
		return out, fmt.Errorf("could not create token: %v", err)
	}

	out.ExpiresAt = time.Now().Add(tokenLifespan)

	return out, nil
}

func (s *Service) deleteVerificationCodeWhenExpires(code string) {
	<-time.After(verificationCodeLifespan)
	if _, err := s.db.Exec("DELETE FROM verification_codes WHERE id = $1", code); err != nil {
		log.Printf("could not delete expired verification code: %v\n", err)
	}
}
