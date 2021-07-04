package nakama

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
)

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
