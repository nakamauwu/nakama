package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
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

func (h *handler) credentialCreationOptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out, data, err := h.svc.CredentialCreationOptions(ctx)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	cookieValue, err := h.cookieCodec.Encode(webAuthnCredentialCreationDataCookieName, data)
	if err != nil {
		h.respondErr(w, fmt.Errorf("could not cookie encode webauthn credential creation data: %w", err))
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

	h.respond(w, out, http.StatusOK)
}

func (h *handler) registerCredential(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(webAuthnCredentialCreationDataCookieName)
	if err == http.ErrNoCookie {
		h.respondErr(w, errWebAuthnTimeout)
		return
	}

	if err != nil {
		h.respondErr(w, fmt.Errorf("could not get webauth credential creation data cookie: %w", err))
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
		h.respondErr(w, errTeaPot)
		return
	}

	reply, err := protocol.ParseCredentialCreationResponse(r)
	if err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	ctx := r.Context()
	err = h.svc.RegisterCredential(ctx, data, reply)
	if err != nil {
		h.respondErr(w, err)
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
	if err != nil {
		h.respondErr(w, err)
		return
	}

	cookieValue, err := h.cookieCodec.Encode(webAuthnCredentialRequestDataCookieName, data)
	if err != nil {
		h.respondErr(w, fmt.Errorf("could not cookie encode webauthn credential request data: %w", err))
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

	h.respond(w, out, http.StatusOK)
}

func (h *handler) webAuthnLogin(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(webAuthnCredentialRequestDataCookieName)
	if err == http.ErrNoCookie {
		h.respondErr(w, errWebAuthnTimeout)
		return
	}

	if err != nil {
		h.respondErr(w, fmt.Errorf("could not get webauth credential creation data cookie: %w", err))
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
		h.respondErr(w, errTeaPot)
		return
	}

	decodedUserID, err := base64.URLEncoding.DecodeString(string(data.UserID))
	if err != nil {
		h.respondErr(w, fmt.Errorf("could not base64 decode session user id: %w", err))
		return
	}

	reply, err := protocol.ParseCredentialRequestResponse(r)
	if err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	ctx := r.Context()
	uid := string(decodedUserID)
	ctx = context.WithValue(ctx, nakama.KeyAuthUserID, uid)
	out, err := h.svc.WebAuthnLogin(ctx, data, reply)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, out, http.StatusOK)
}
