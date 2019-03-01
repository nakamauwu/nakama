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
const KeyAuthUserID key = "auth_user_id"

const (
	verificationCodeLifespan = time.Minute * 15
	tokenLifespan            = time.Hour * 24 * 14
)

var (
	// ErrUnimplemented denotes that the method is not implemented.
	ErrUnimplemented = errors.New("unimplemented")
	// ErrUnauthenticated denotes no authenticated user in context.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrInvalidRedirectURI denotes that the given redirect uri was not valid.
	ErrInvalidRedirectURI = errors.New("invalid redirect uri")
	// ErrInvalidVerificationCode denotes that the given verification code is not valid.
	ErrInvalidVerificationCode = errors.New("invalid verification code")
	// ErrVerificationCodeNotFound denotes that the verification code was not found.
	ErrVerificationCodeNotFound = errors.New("verification code not found")
	// ErrVerificationCodeExpired denotes that the verification code is already expired.
	ErrVerificationCodeExpired = errors.New("verification code expired")
)

var rxUUID = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

var magicLinkMailTmpl *template.Template

type key string

// LoginOutput response.
type LoginOutput struct {
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

	var verificationCode string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO verification_codes (user_id) VALUES (
			(SELECT id FROM users WHERE email = $1)
		) RETURNING id`, email).Scan(&verificationCode)
	if isForeignKeyViolation(err) {
		return ErrUserNotFound
	}

	if err != nil {
		return fmt.Errorf("could not insert verification code: %v", err)
	}

	magicLink := s.origin
	magicLink.Path = "/api/auth_redirect"
	q := magicLink.Query()
	q.Set("verification_code", verificationCode)
	q.Set("redirect_uri", uri.String())
	magicLink.RawQuery = q.Encode()

	if magicLinkMailTmpl == nil {
		magicLinkMailTmpl, err = template.ParseFiles("web/template/mail/magic-link.html")
		if err != nil {
			return fmt.Errorf("could not parse magic link mail template: %v", err)
		}
	}

	var mail bytes.Buffer
	if err = magicLinkMailTmpl.Execute(&mail, map[string]interface{}{
		"MagicLink": magicLink.String(),
		"Minutes":   int(verificationCodeLifespan.Minutes()),
	}); err != nil {
		return fmt.Errorf("could not execute magic link mail template: %v", err)
	}

	if err = s.sendMail(email, "Magic Link", mail.String()); err != nil {
		return fmt.Errorf("could not send magic link: %v", err)
	}

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
	var ts time.Time
	err = s.db.QueryRowContext(ctx, `
		DELETE FROM verification_codes WHERE id = $1
		RETURNING user_id, created_at`, verificationCode).Scan(&uid, &ts)
	if err == sql.ErrNoRows {
		return "", ErrVerificationCodeNotFound
	}

	if err != nil {
		return "", fmt.Errorf("could not delete verification code: %v", err)
	}

	if ts.Add(verificationCodeLifespan).Before(time.Now()) {
		return "", ErrVerificationCodeExpired
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

// Login insecurely. For development purposes only.
func (s *Service) Login(ctx context.Context, email string) (LoginOutput, error) {
	var out LoginOutput

	if s.origin.Hostname() == "localhost" {
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

// AuthUserID from token.
func (s *Service) AuthUserID(token string) (int64, error) {
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

// AuthUser from context.
// It requires the user ID in the context, so add it with a middleware or something.
func (s *Service) AuthUser(ctx context.Context) (User, error) {
	var u User
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return u, ErrUnauthenticated
	}

	return s.userByID(ctx, uid)
}

func (s *Service) deleteExpiredVerificationCodesCronJob(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour * 24):
			if _, err := s.db.ExecContext(ctx,
				fmt.Sprintf(`DELETE FROM verification_codes WHERE created_at < now() - INTERVAL '%dm'`,
					int(verificationCodeLifespan.Minutes()))); err != nil {
				log.Printf("could not delete expired verification codes: %v\n", err)
			}
		}
	}
}
