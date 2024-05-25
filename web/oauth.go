package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/nakamauwu/nakama/oauth"
	"github.com/nakamauwu/nakama/service"
	"github.com/nakamauwu/nakama/types"
	"github.com/nicolasparada/go-errs"
)

const sessKeyOAuth2Username = "oauth2_username"

var errOAuth2StateMismatch = errors.New("oauth2 state mismatch")

func sessKeyOAuth2State(providerName string) string {
	return fmt.Sprintf("oauth2_state_%s", providerName)
}

// providerRedirect handles GET /oauth2/{provider}/redirect
func (h *Handler) providerRedirect(provider oauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := genRandStr()
		if err != nil {
			h.renderErr(w, r, fmt.Errorf("generate oauth state: %w", err))
			return
		}

		ctx := r.Context()
		h.sess.Put(ctx, sessKeyOAuth2State(provider.Name()), state)

		q := r.URL.Query()
		if q.Has("username") {
			h.sess.Put(ctx, sessKeyOAuth2Username, q.Get("username"))
		}

		http.Redirect(w, r, provider.AuthCodeURL(state), http.StatusFound)
	}
}

// providerCallback handles GET /oauth2/{provider}/callback
func (h *Handler) providerCallback(provider oauth.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		expectedState := h.sess.PopString(ctx, sessKeyOAuth2State(provider.Name()))
		receivedState := q.Get("state")
		if receivedState == "" || receivedState != expectedState {
			h.renderErr(w, r, errOAuth2StateMismatch)
			return
		}

		token, err := provider.Exchange(ctx, q.Get("code"))
		if err != nil {
			h.renderErr(w, r, err)
			return
		}

		claims, err := provider.Claims(ctx, token)
		if err != nil {
			h.renderErr(w, r, err)
			return
		}

		if !claims.EmailVerified {
			h.renderErr(w, r, errs.UnauthenticatedError("email not verified"))
			return
		}

		login := types.Login{
			Email: claims.Email,
		}

		if h.sess.Exists(ctx, sessKeyOAuth2Username) {
			login.Username = ptr(h.sess.PopString(ctx, sessKeyOAuth2Username))
		}

		user, err := h.Service.Login(ctx, login)
		if errors.Is(err, service.ErrUsernameRequired) {
			h.render(w, r, "register.tmpl", map[string]any{
				"Session":      h.sessionData(r),
				"Email":        claims.Email,
				"ProviderName": provider.Name(),
			}, http.StatusOK)
			return
		}

		if err != nil {
			h.renderErr(w, r, err)
			return
		}

		if err := h.sess.RenewToken(ctx); err != nil {
			h.renderErr(w, r, err)
			return
		}

		h.sess.Put(ctx, sessKeyUserID, user.ID)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
