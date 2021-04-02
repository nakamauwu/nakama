package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/nicolasparada/nakama/internal/service"
)

const WebAuthnTimeout = time.Minute * 2
const (
	webAuthnCredentialCreationDataCookieName = "webauthn_credential_creation_data"
	webAuthnCredentialRequestDataCookieName  = "webauthn_credential_request_data"
)

type loginInput struct {
	Email string
}

type sendMagicLinkInput struct {
	Email       string
	RedirectURI string
}

func (h *handler) sendMagicLink(w http.ResponseWriter, r *http.Request) {
	var in sendMagicLinkInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err := h.SendMagicLink(r.Context(), in.Email, in.RedirectURI)
	if err == service.ErrInvalidEmail || err == service.ErrInvalidRedirectURI {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) authRedirect(w http.ResponseWriter, r *http.Request) {
	uri, err := h.AuthURI(r.Context(), r.RequestURI)
	if err == service.ErrInvalidRedirectURI {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	http.Redirect(w, r, uri.String(), http.StatusFound)
}

func (h *handler) createCredentialCreationOptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u, err := h.AuthUser(ctx)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	webauthnUser := service.WebAuthnUser{User: u}
	out, data, err := h.webauthn.BeginRegistration(webauthnUser,
		webauthn.WithAuthenticatorSelection(webauthn.SelectAuthenticator(
			string(protocol.Platform),
			nil,
			string(protocol.VerificationRequired),
		)),
	)
	if err != nil {
		respondErr(w, fmt.Errorf("could not begin webauthn registration: %w", err))
		return
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		respondErr(w, fmt.Errorf("could not json marshall credential creation data: %w", err))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialCreationDataCookieName,
		Value:    base64.URLEncoding.EncodeToString(rawData),
		MaxAge:   int(WebAuthnTimeout.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	respond(w, out, http.StatusOK)
}

func (h *handler) registerCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u, err := h.AuthUser(ctx)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	c, err := r.Cookie(webAuthnCredentialCreationDataCookieName)
	if err == http.ErrNoCookie {
		http.Error(w, "webAuthn timeout", http.StatusBadRequest)
		return
	}

	if err != nil {
		respondErr(w, fmt.Errorf("could not get webauth credential creation data cookie: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialCreationDataCookieName,
		Value:    "",
		MaxAge:   -1,
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	rawData, err := base64.URLEncoding.DecodeString(c.Value)
	if err != nil {
		respondErr(w, fmt.Errorf("could not base64 decode credential creation data: %w", err))
		return
	}

	var data webauthn.SessionData
	err = json.Unmarshal(rawData, &data)
	if err != nil {
		respondErr(w, fmt.Errorf("could not json unmarshal credential creation data: %w", err))
		return
	}

	webauthnUser := service.WebAuthnUser{User: u}
	out, err := h.webauthn.FinishRegistration(webauthnUser, data, r)
	if err != nil {
		respondErr(w, fmt.Errorf("could not finish webauthn registration: %w", err))
		return
	}

	err = h.CreateCredential(ctx, out)
	if err == service.ErrWebAuthnCredentialExists {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) createCredentialRequestOptions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	email := strings.TrimSpace(q.Get("email"))

	ctx := r.Context()
	u, err := h.WebAuthnUser(ctx, service.WebAuthnUserByEmail(email))
	if err == service.ErrInvalidEmail {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	if len(u.WebAuthnCredentials()) == 0 {
		http.Error(w, "no credentials", http.StatusBadRequest)
		return
	}

	var opts []webauthn.LoginOption
	if s := strings.TrimSpace(q.Get("credential_id")); s != "" {
		credentialID, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			http.Error(w, "invalid webauthn credential ID", http.StatusUnprocessableEntity)
			return
		}

		allowList := []protocol.CredentialDescriptor{{
			CredentialID: credentialID,
			Type:         protocol.CredentialType("public-key"),
		}}
		opts = append(opts, webauthn.WithAllowedCredentials(allowList))
	}
	out, data, err := h.webauthn.BeginLogin(u, opts...)
	if err != nil {
		respondErr(w, fmt.Errorf("could not begin webauth login: %w", err))
		return
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		respondErr(w, fmt.Errorf("could not json marshal credential request data: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialRequestDataCookieName,
		Value:    base64.URLEncoding.EncodeToString(rawData),
		MaxAge:   int(WebAuthnTimeout.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	respond(w, out, http.StatusOK)
}

func (h *handler) webAuthnLogin(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(webAuthnCredentialRequestDataCookieName)
	if err == http.ErrNoCookie {
		http.Error(w, "webAuthn timeout", http.StatusBadRequest)
		return
	}

	if err != nil {
		respondErr(w, fmt.Errorf("could not get webauth credential creation data cookie: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialRequestDataCookieName,
		Value:    "",
		MaxAge:   -1,
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	rawData, err := base64.URLEncoding.DecodeString(c.Value)
	if err != nil {
		respondErr(w, fmt.Errorf("could not base64 decode credential creation data: %w", err))
		return
	}

	var data webauthn.SessionData
	err = json.Unmarshal(rawData, &data)
	if err != nil {
		respondErr(w, fmt.Errorf("could not json unmarshal credential creation data: %w", err))
		return
	}

	decodedUserID, err := base64.URLEncoding.DecodeString(string(data.UserID))
	if err != nil {
		respondErr(w, fmt.Errorf("could not base64  decode session user id: %w", err))
		return
	}

	ctx := r.Context()
	uid := string(decodedUserID)
	ctx = context.WithValue(ctx, service.KeyAuthUserID, uid)
	u, err := h.WebAuthnUser(ctx)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	cred, err := h.webauthn.FinishLogin(u, data, r)
	if err != nil {
		respondErr(w, fmt.Errorf("could not finish webauth login: %w", err))
		return
	}

	if cred.Authenticator.CloneWarning {
		http.Error(w, "credential cloned", http.StatusTeapot)
		return
	}

	err = h.UpdateWebAuthnAuthenticatorSignCount(ctx, cred.ID, cred.Authenticator.SignCount)
	if err != nil {
		respondErr(w, err)
		return
	}

	tokenOutput, err := h.Token(ctx)
	if err != nil {
		respondErr(w, err)
		return
	}

	out := service.DevLoginOutput{
		User:      u.User,
		Token:     tokenOutput.Token,
		ExpiresAt: tokenOutput.ExpiresAt,
	}

	respond(w, out, http.StatusOK)
}

func (h *handler) devLogin(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.DevLogin(r.Context(), in.Email)
	if err == service.ErrUnimplemented {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	if err == service.ErrInvalidEmail {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, out, http.StatusOK)
}

func (h *handler) authUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.AuthUser(r.Context())
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, u, http.StatusOK)
}

func (h *handler) token(w http.ResponseWriter, r *http.Request) {
	out, err := h.Token(r.Context())
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, out, http.StatusOK)
}

func (h *handler) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.URL.Query().Get("auth_token"))

		if token == "" {
			if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
				token = a[7:]
			}
		}

		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		uid, err := h.AuthUserIDFromToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, service.KeyAuthUserID, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
