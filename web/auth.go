package web

import (
	"net/http"

	"github.com/nakamauwu/nakama/auth"
	"github.com/nakamauwu/nakama/web/templates"
)

const sessKeyUserID = "user_id"

// showLogin handles GET /login
func (h *Handler) showLogin(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, templates.Login(), http.StatusOK)
}

func (h *Handler) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if !h.sess.Exists(ctx, sessKeyUserID) {
			next.ServeHTTP(w, r)
			return
		}

		userID := h.sess.GetString(ctx, sessKeyUserID)

		user, err := h.Service.User(ctx, userID)
		if err != nil {
			h.renderErr(w, r, err)
			return
		}

		ctx = auth.ContextWithUser(ctx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// logout handles POST /logout
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.sess.RenewToken(ctx); err != nil {
		h.renderErr(w, r, err)
		return
	}

	if err := h.sess.Clear(ctx); err != nil {
		h.renderErr(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
