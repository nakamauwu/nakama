package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/nicolasparada/nakama"
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

	err := h.svc.SendMagicLink(r.Context(), in.Email, in.RedirectURI)
	if err == nakama.ErrInvalidEmail || err == nakama.ErrInvalidRedirectURI {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrUntrustedRedirectURI {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	if err == nakama.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func emptyStringPtr(s string) *string {
	if s != "" {
		return &s
	}

	return nil
}

func (h *handler) verifyMagicLink(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI, err := h.svc.ParseRedirectURI(q.Get("redirect_uri"))
	if err == nakama.ErrInvalidRedirectURI {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrUntrustedRedirectURI {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	auth, err := h.svc.VerifyMagicLink(r.Context(), q.Get("email"), q.Get("verification_code"), emptyStringPtr(q.Get("username")))
	if err == nakama.ErrUserNotFound || err == nakama.ErrUsernameTaken {
		redirectWithHashFragment(w, r, redirectURI, url.Values{
			"error":          []string{err.Error()},
			"retry_endpoint": []string{r.RequestURI},
		}, http.StatusFound)
		return
	}

	if err == nakama.ErrInvalidEmail ||
		err == nakama.ErrInvalidVerificationCode ||
		err == nakama.ErrInvalidUsername ||
		err == nakama.ErrVerificationCodeNotFound ||
		err == nakama.ErrExpiredToken ||
		err == nakama.ErrEmailTaken {
		redirectWithHashFragment(w, r, redirectURI, url.Values{
			"error": []string{err.Error()},
		}, http.StatusFound)
		return
	}

	if err != nil {
		log.Println(err)
		redirectWithHashFragment(w, r, redirectURI, url.Values{
			"error": []string{"internal server error"},
		}, http.StatusFound)
		return
	}

	values := url.Values{
		"token":         []string{auth.Token},
		"expires_at":    []string{auth.ExpiresAt.Format(time.RFC3339Nano)},
		"user.id":       []string{auth.User.ID},
		"user.username": []string{auth.User.Username},
	}
	if auth.User.AvatarURL != nil {
		values.Set("user.avatar_url", *auth.User.AvatarURL)
	}
	redirectWithHashFragment(w, r, redirectURI, values, http.StatusFound)
}

func redirectWithHashFragment(w http.ResponseWriter, r *http.Request, uri *url.URL, data url.Values, statusCode int) {
	// Using query string instead of hash fragment because golang's url.URL#RawFragment is a no-op.
	// We set the RawQuery instead, and then string replace the "?" symbol by "#".
	uri.RawQuery = data.Encode()
	location := uri.String()
	location = strings.Replace(location, "?", "#", 1)
	http.Redirect(w, r, location, statusCode)
}

func (h *handler) credentialCreationOptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out, data, err := h.svc.CredentialCreationOptions(ctx)
	if err == nakama.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == nakama.ErrUserGone {
		http.Error(w, err.Error(), http.StatusGone)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	cookieValue, err := h.cookieCodec.Encode(webAuthnCredentialCreationDataCookieName, data)
	if err != nil {
		respondErr(w, fmt.Errorf("could not cookie encode webauthn credential creation data: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialCreationDataCookieName,
		Value:    cookieValue,
		MaxAge:   int(WebAuthnTimeout.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	respond(w, out, http.StatusOK)
}

func (h *handler) registerCredential(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(webAuthnCredentialCreationDataCookieName)
	if err == http.ErrNoCookie {
		http.Error(w, "webAuthn timeout", http.StatusBadRequest)
		return
	}

	if err != nil {
		respondErr(w, fmt.Errorf("could not get webauth credential creation data cookie: %w", err))
		return
	}

	cookieValue := c.Value
	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialCreationDataCookieName,
		Value:    "",
		MaxAge:   -1,
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	var data webauthn.SessionData
	err = h.cookieCodec.Decode(webAuthnCredentialCreationDataCookieName, cookieValue, &data)
	if err != nil {
		http.Error(w, "i am a teapot", http.StatusTeapot)
		return
	}

	reply, err := protocol.ParseCredentialCreationResponse(r)
	if err != nil {
		respond(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err = h.svc.RegisterCredential(ctx, data, reply)
	if err == nakama.ErrWebAuthnCredentialExists {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) credentialRequestOptions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	email := strings.TrimSpace(q.Get("email"))
	var opts []nakama.CredentialRequestOptionsOpt
	if s := strings.TrimSpace(q.Get("credential_id")); s != "" {
		opts = append(opts, nakama.CredentialRequestOptionsWithCredentialID(s))
	}

	ctx := r.Context()
	out, data, err := h.svc.CredentialRequestOptions(ctx, email, opts...)
	if err == nakama.ErrInvalidEmail || err == nakama.ErrInvalidWebAuthnCredentialID {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err == nakama.ErrNoWebAuthnCredentials {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	cookieValue, err := h.cookieCodec.Encode(webAuthnCredentialRequestDataCookieName, data)
	if err != nil {
		respondErr(w, fmt.Errorf("could not cookie encode webauthn credential request data: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialRequestDataCookieName,
		Value:    cookieValue,
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

	cookieValue := c.Value

	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCredentialRequestDataCookieName,
		Value:    "",
		MaxAge:   -1,
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	var data webauthn.SessionData
	err = h.cookieCodec.Decode(webAuthnCredentialRequestDataCookieName, cookieValue, &data)
	if err != nil {
		http.Error(w, "i am a teapot", http.StatusTeapot)
		return
	}

	decodedUserID, err := base64.URLEncoding.DecodeString(string(data.UserID))
	if err != nil {
		respondErr(w, fmt.Errorf("could not base64 decode session user id: %w", err))
		return
	}

	reply, err := protocol.ParseCredentialRequestResponse(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	uid := string(decodedUserID)
	ctx = context.WithValue(ctx, nakama.KeyAuthUserID, uid)
	out, err := h.svc.WebAuthnLogin(ctx, data, reply)
	if err == nakama.ErrUnauthenticated || err == nakama.ErrInvalidWebAuthnCredentials {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == nakama.ErrUserGone {
		http.Error(w, err.Error(), http.StatusGone)
		return
	}

	if err == nakama.ErrWebAuthnCredentialCloned {
		http.Error(w, err.Error(), http.StatusTeapot)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
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

	out, err := h.svc.DevLogin(r.Context(), in.Email)
	if err == nakama.ErrUnimplemented {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	if err == nakama.ErrInvalidEmail {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrUserNotFound {
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
	u, err := h.svc.AuthUser(r.Context())
	if err == nakama.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == nakama.ErrUserNotFound {
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
	out, err := h.svc.Token(r.Context())
	if err == nakama.ErrUnauthenticated {
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

		uid, err := h.svc.AuthUserIDFromToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, nakama.KeyAuthUserID, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
