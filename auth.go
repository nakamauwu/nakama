package nakama

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/hako/branca"
	"github.com/hako/durafmt"
	"github.com/nicolasparada/nakama/web"
)

// KeyAuthUserID to use in context.
const KeyAuthUserID = ctxkey("auth_user_id")
const WebAuthnTimeout = time.Minute * 2

const (
	emailVerificationCodeTTL = time.Hour * 2
	authTokenTTL             = time.Hour * 24 * 14
)

var (
	// ErrInvalidRedirectURI denotes an invalid redirect URI.
	ErrInvalidRedirectURI = InvalidArgumentError("invalid redirect URI")
	// ErrUntrustedRedirectURI denotes an untrusted redirect URI.
	// That is an URI that is not in the same host as the nakama.
	ErrUntrustedRedirectURI = PermissionDeniedError("untrusted redirect URI")
	// ErrInvalidToken denotes an invalid token.
	ErrInvalidToken = InvalidArgumentError("invalid token")
	// ErrExpiredToken denotes that the token already expired.
	ErrExpiredToken = UnauthenticatedError("expired token")
	// ErrInvalidVerificationCode denotes an invalid verification code.
	ErrInvalidVerificationCode = InvalidArgumentError("invalid verification code")
	// ErrVerificationCodeNotFound denotes a not found verification code.
	ErrVerificationCodeNotFound = NotFoundError("verification code not found")
	// ErrWebAuthnCredentialExists denotes that the webauthn credential ID already exists for the given user.
	ErrWebAuthnCredentialExists = AlreadyExistsError("webAuthn credential exists")
	// ErrNoWebAuthnCredentials denotes that the user has no registered webauthn credentials yet.
	ErrNoWebAuthnCredentials = NotFoundError("no webAuthn credentials")
	// ErrInvalidWebAuthnCredentialID denotes an invalid webauthn credential ID.
	ErrInvalidWebAuthnCredentialID = InvalidArgumentError("invalid webAuthn credential ID")
	// ErrInvalidWebAuthnCredentials denotes invalid webauthn credentials.
	ErrInvalidWebAuthnCredentials = InvalidArgumentError("invalid webAuthn credentials")
	// ErrWebAuthnCredentialCloned denotes that the webauthn credential may be cloned.
	ErrWebAuthnCredentialCloned = AlreadyExistsError("webAuthn credential cloned")
)

type ctxkey string

// TokenOutput response.
type TokenOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// AuthOutput response.
type AuthOutput struct {
	User      User      `json:"user"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type webAuthnUser struct {
	User        User
	Credentials []webauthn.Credential
}

func (u webAuthnUser) WebAuthnID() []byte {
	return []byte(base64.URLEncoding.EncodeToString([]byte(u.User.ID)))
}

func (u webAuthnUser) WebAuthnName() string {
	return u.User.Username
}

func (u webAuthnUser) WebAuthnDisplayName() string {
	return u.User.Username
}

func (u webAuthnUser) WebAuthnIcon() string {
	if u.User.AvatarURL == nil {
		return ""
	}
	return *u.User.AvatarURL
}

func (u webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// SendMagicLink to login without passwords.
// A second endpoint GET /api/verify_magic_link?email&code&redirect_uri must exist.
func (s *Service) SendMagicLink(ctx context.Context, email, redirectURI string) error {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	if !reEmail.MatchString(email) {
		return ErrInvalidEmail
	}

	_, err := s.ParseRedirectURI(redirectURI)
	if err != nil {
		return err
	}

	var code string
	query := "INSERT INTO email_verification_codes (email) VALUES ($1) RETURNING code"
	row := s.DB.QueryRowContext(ctx, query, email)
	err = row.Scan(&code)
	if err != nil {
		return fmt.Errorf("could not insert verification code: %w", err)
	}

	defer func() {
		if err != nil {
			go func() {
				query := "DELETE FROM email_verification_codes WHERE email = $1 AND code = $2"
				_, err := s.DB.Exec(query, email, code)
				if err != nil {
					_ = s.Logger.Log("error", fmt.Errorf("could not delete verification code: %w", err))
				}
			}()
		}
	}()

	// See transport/http/handler.go
	// GET /api/verify_magic_link must exist.
	magicLink := cloneURL(s.Origin)
	magicLink.Path = "/api/verify_magic_link"
	q := magicLink.Query()
	q.Set("email", email)
	q.Set("verification_code", code)
	q.Set("redirect_uri", redirectURI)
	magicLink.RawQuery = q.Encode()

	s.magicLinkTmplOncer.Do(func() {
		var text []byte
		text, err = web.Files.ReadFile("template/mail/magic-link.html.tmpl")
		if err != nil {
			err = fmt.Errorf("could not read magic link template file: %w", err)
			return
		}

		s.magicLinkTmpl, err = template.
			New("mail/magic-link.html").
			Funcs(template.FuncMap{
				"human_duration": func(d time.Duration) string {
					return durafmt.Parse(d).LimitFirstN(1).String()
				},
				"html": func(s string) template.HTML {
					return template.HTML(s)
				},
			}).
			Parse(string(text))
		if err != nil {
			err = fmt.Errorf("could not parse magic link mail template: %w", err)
			return
		}
	})
	if err != nil {
		return err
	}

	var b bytes.Buffer
	err = s.magicLinkTmpl.Execute(&b, map[string]interface{}{
		"Origin":    s.Origin,
		"MagicLink": magicLink,
		"TTL":       emailVerificationCodeTTL,
	})
	if err != nil {
		return fmt.Errorf("could not execute magic link mail template: %w", err)
	}

	err = s.Sender.Send(email, "Login to Nakama", b.String(), magicLink.String())
	if err != nil {
		return fmt.Errorf("could not send magic link: %w", err)
	}

	return nil
}

// ParseRedirectURI the given redirect URI and validates it.
func (s *Service) ParseRedirectURI(rawurl string) (*url.URL, error) {
	uri, err := url.Parse(rawurl)
	if err != nil || !uri.IsAbs() {
		return nil, ErrInvalidRedirectURI
	}

	if uri.Host != s.Origin.Host {
		return nil, ErrUntrustedRedirectURI
	}

	return uri, nil
}

// VerifyMagicLink checks whether the given email and verification code exists and issues a new auth token.
// If the user does not exists, it can create a new one with the given username.
func (s *Service) VerifyMagicLink(ctx context.Context, email, code string, username *string) (AuthOutput, error) {
	var auth AuthOutput

	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	if !reEmail.MatchString(email) {
		return auth, ErrInvalidEmail
	}

	if !reUUID.MatchString(code) {
		return auth, ErrInvalidVerificationCode
	}

	if username != nil && !reUsername.MatchString(*username) {
		return auth, ErrInvalidUsername
	}

	err := crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		var createdAt time.Time
		query := "SELECT created_at FROM email_verification_codes WHERE email = $1 AND code = $2"
		row := tx.QueryRowContext(ctx, query, email, code)
		err := row.Scan(&createdAt)
		if err == sql.ErrNoRows {
			return ErrVerificationCodeNotFound
		}

		if err != nil {
			return fmt.Errorf("could not sql query select verification code: %w", err)
		}

		if isVerificationCodeExpired(createdAt) {
			return ErrExpiredToken
		}

		var avatar sql.NullString
		query = "SELECT id, username, avatar FROM users WHERE email = $1"
		row = tx.QueryRowContext(ctx, query, email)
		err = row.Scan(&auth.User.ID, &auth.User.Username, &avatar)
		if err == sql.ErrNoRows {
			if username == nil {
				return ErrUserNotFound
			}

			query := "INSERT INTO users (email, username) VALUES ($1, $2) RETURNING id"
			row := tx.QueryRowContext(ctx, query, email, username)
			err := row.Scan(&auth.User.ID)
			if isUniqueViolation(err) {
				if strings.Contains(err.Error(), "email") {
					return ErrEmailTaken
				}

				if strings.Contains(err.Error(), "username") {
					return ErrUsernameTaken
				}
			}

			if err != nil {
				return fmt.Errorf("could not sql insert user at magic link: %w", err)
			}

			auth.User.Username = *username

			return nil
		}

		if err != nil {
			return fmt.Errorf("could not sql query select user from verification code email: %w", err)
		}

		auth.User.AvatarURL = s.avatarURL(avatar)

		return nil
	})
	if err != nil {
		return auth, err
	}

	auth.ExpiresAt = time.Now().Add(authTokenTTL)
	auth.Token, err = s.codec().EncodeToString(auth.User.ID)
	if err != nil {
		return auth, fmt.Errorf("could not create auth token: %w", err)
	}

	go func() {
		_, err := s.DB.Exec("DELETE FROM email_verification_codes WHERE email = $1 AND code = $2", email, code)
		if err != nil {
			_ = s.Logger.Log("error", fmt.Errorf("could not delete verification code: %w", err))
			return
		}
	}()

	return auth, nil
}

func isVerificationCodeExpired(t time.Time) bool {
	now := time.Now()
	exp := t.Add(emailVerificationCodeTTL)
	return exp.Equal(now) || exp.Before(now)
}

func (s *Service) CredentialCreationOptions(ctx context.Context) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	u, err := s.webAuthnUser(ctx)
	if err != nil {
		return nil, nil, err
	}

	excludedCredentials := make([]protocol.CredentialDescriptor, len(u.Credentials))
	for i, cred := range u.Credentials {
		excludedCredentials[i].CredentialID = cred.ID
		excludedCredentials[i].Type = protocol.CredentialType("public-key")
		excludedCredentials[i].Transport = []protocol.AuthenticatorTransport{
			protocol.USB,
			protocol.NFC,
			protocol.BLE,
			protocol.Internal,
		}
	}
	return s.WebAuthn.BeginRegistration(u,
		webauthn.WithAuthenticatorSelection(webauthn.SelectAuthenticator(
			string(protocol.Platform),
			nil,
			string(protocol.VerificationRequired),
		)),
		webauthn.WithExclusions(excludedCredentials),
	)
}

func (s *Service) RegisterCredential(ctx context.Context, data webauthn.SessionData, reply *protocol.ParsedCredentialCreationData) error {
	u, err := s.webAuthnUser(ctx)
	if err != nil {
		return err
	}

	cred, err := s.WebAuthn.CreateCredential(u, data, reply)
	if err != nil {
		return fmt.Errorf("could not create webauthn credential: %w", err)
	}

	return crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		query := `
			INSERT INTO webauthn_authenticators (
				aaguid,
				sign_count,
				clone_warning
			) VALUES ($1, $2, $3)
			RETURNING id
		`
		row := tx.QueryRowContext(ctx, query,
			cred.Authenticator.AAGUID,
			cred.Authenticator.SignCount,
			cred.Authenticator.CloneWarning,
		)
		var authenticatorID string
		err := row.Scan(&authenticatorID)
		if err != nil {
			return fmt.Errorf("could not sql insert and scan webauthn authenticator id: %w", err)
		}

		query = `
			INSERT INTO webauthn_credentials (
				webauthn_authenticator_id,
				user_id,
				credential_id,
				public_key,
				attestation_type
			) VALUES ($1, $2, $3, $4, $5)
		`
		_, err = tx.ExecContext(ctx, query,
			authenticatorID,
			u.User.ID,
			base64.URLEncoding.EncodeToString(cred.ID),
			cred.PublicKey,
			cred.AttestationType,
		)
		if isUniqueViolation(err) {
			return ErrWebAuthnCredentialExists
		}

		if err != nil {
			return fmt.Errorf("could not sql insert webauthn credential: %w", err)
		}

		return nil
	})
}

type CredentialRequestOptionsOpts struct {
	CredentialID *string
}

type CredentialRequestOptionsOpt func(*CredentialRequestOptionsOpts)

func CredentialRequestOptionsWithCredentialID(credentialID string) CredentialRequestOptionsOpt {
	return func(opts *CredentialRequestOptionsOpts) {
		opts.CredentialID = &credentialID
	}
}

func (s *Service) CredentialRequestOptions(ctx context.Context, email string, opts ...CredentialRequestOptionsOpt) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	var options CredentialRequestOptionsOpts
	for _, o := range opts {
		o(&options)
	}

	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	if !reEmail.MatchString(email) {
		return nil, nil, ErrInvalidEmail
	}

	u, err := s.webAuthnUser(ctx, webAuthnUserByEmail(email))
	if err != nil {
		return nil, nil, err
	}

	if len(u.Credentials) == 0 {
		return nil, nil, ErrNoWebAuthnCredentials
	}

	var loginOpts []webauthn.LoginOption
	if options.CredentialID != nil {
		credentialID, err := base64.RawURLEncoding.DecodeString(*options.CredentialID)
		if err != nil {
			return nil, nil, ErrInvalidWebAuthnCredentialID
		}

		loginOpts = append(loginOpts, webauthn.WithAllowedCredentials(
			[]protocol.CredentialDescriptor{{
				CredentialID: credentialID,
				Type:         protocol.CredentialType("public-key"),
				Transport: []protocol.AuthenticatorTransport{
					protocol.USB,
					protocol.NFC,
					protocol.BLE,
					protocol.Internal,
				},
			}},
		))
	}
	out, data, err := s.WebAuthn.BeginLogin(u, loginOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("could not begin webauthn login: %w", err)
	}

	return out, data, nil
}

type webAuthnUserOpts struct {
	Email *string
}

type webAuthnUserOpt func(*webAuthnUserOpts)

func webAuthnUserByEmail(email string) webAuthnUserOpt {
	return func(opts *webAuthnUserOpts) {
		opts.Email = &email
	}
}

func (s *Service) webAuthnUser(ctx context.Context, opts ...webAuthnUserOpt) (webAuthnUser, error) {
	var u webAuthnUser
	var options webAuthnUserOpts
	for _, o := range opts {
		o(&options)
	}

	data := map[string]interface{}{}
	if options.Email != nil {
		*options.Email = strings.ToLower(*options.Email)
		if !reEmail.MatchString(*options.Email) {
			return u, ErrInvalidEmail
		}

		data["field"] = "users.email"
		data["value"] = *options.Email
	} else {
		uid, ok := ctx.Value(KeyAuthUserID).(string)
		if !ok {
			return u, ErrUnauthenticated
		}

		data["field"] = "users.id"
		data["value"] = uid
	}

	userQuery, userArgs, err := buildQuery(`
		SELECT id, username, avatar FROM users WHERE {{ .field }} = @value
	`, data)
	if err != nil {
		return u, fmt.Errorf("could not build webauthn user sql query: %w", err)
	}

	err = crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		var avatar sql.NullString
		row := tx.QueryRowContext(ctx, userQuery, userArgs...)
		err := row.Scan(&u.User.ID, &u.User.Username, &avatar)
		if err == sql.ErrNoRows {
			if options.Email != nil {
				return ErrUserNotFound
			}

			return ErrUserGone
		}

		if err != nil {
			return fmt.Errorf("could not sql select webauthn user: %w", err)
		}

		u.User.AvatarURL = s.avatarURL(avatar)

		query := `
			SELECT
				webauthn_credentials.credential_id,
				webauthn_credentials.public_key,
				webauthn_credentials.attestation_type,
				webauthn_authenticators.aaguid,
				webauthn_authenticators.sign_count,
				webauthn_authenticators.clone_warning
			FROM webauthn_credentials
			INNER JOIN webauthn_authenticators
			ON webauthn_credentials.webauthn_authenticator_id = webauthn_authenticators.id
			WHERE webauthn_credentials.user_id = $1
		`
		rows, err := tx.QueryContext(ctx, query, u.User.ID)
		if err != nil {
			return fmt.Errorf("could not sql query select webauthn credentials: %w", err)
		}

		defer rows.Close()

		u.Credentials = nil
		for rows.Next() {
			var cred webauthn.Credential
			var credentialID string
			err := rows.Scan(
				&credentialID,
				&cred.PublicKey,
				&cred.AttestationType,
				&cred.Authenticator.AAGUID,
				&cred.Authenticator.SignCount,
				&cred.Authenticator.CloneWarning,
			)
			if err != nil {
				return fmt.Errorf("could not sql scan webauthn credential: %w", err)
			}

			cred.ID, err = base64.URLEncoding.DecodeString(credentialID)
			if err != nil {
				return fmt.Errorf("could not base64 decode webauthn credential id: %w", err)
			}

			u.Credentials = append(u.Credentials, cred)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("could not not iterate over webauthn credentials: %w", err)
		}

		return nil
	})
	return u, err
}

func (s *Service) WebAuthnLogin(ctx context.Context, data webauthn.SessionData, reply *protocol.ParsedCredentialAssertionData) (AuthOutput, error) {
	var out AuthOutput
	u, err := s.webAuthnUser(ctx)
	if err != nil {
		return out, err
	}

	cred, err := s.WebAuthn.ValidateLogin(u, data, reply)
	if err != nil {
		return out, ErrInvalidWebAuthnCredentials
	}

	if cred.Authenticator.CloneWarning {
		return out, ErrWebAuthnCredentialCloned
	}

	query := `
		UPDATE webauthn_authenticators SET sign_count = $1
		WHERE id = (
			SELECT webauthn_authenticator_id FROM webauthn_credentials WHERE credential_id = $2
		)
	`
	_, err = s.DB.ExecContext(ctx, query,
		cred.Authenticator.SignCount,
		base64.URLEncoding.EncodeToString(cred.ID),
	)
	if err != nil {
		return out, fmt.Errorf("could not sql update webauthn authenticator sign count: %w", err)
	}

	tokenOutput, err := s.Token(ctx)
	if err != nil {
		return out, err
	}

	out.User = u.User
	out.Token = tokenOutput.Token
	out.ExpiresAt = tokenOutput.ExpiresAt
	return out, nil
}

// DevLogin is a login for development purposes only.
// TODO: disable dev login on production.
func (s *Service) DevLogin(ctx context.Context, email string) (AuthOutput, error) {
	var out AuthOutput

	if s.DisabledDevLogin {
		return out, ErrUnimplemented
	}

	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
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

	out.ExpiresAt = time.Now().Add(authTokenTTL)

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

	out.ExpiresAt = time.Now().Add(authTokenTTL)

	return out, nil
}

func (s *Service) codec() *branca.Branca {
	cdc := branca.NewBranca(s.TokenKey)
	cdc.SetTTL(uint32(authTokenTTL.Seconds()))
	return cdc
}
