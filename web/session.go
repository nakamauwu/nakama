package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
)

type Session struct {
	IsLoggedIn bool
	User       nakama.User
}

func (h *Handler) sessionFromReq(r *http.Request) Session {
	var out Session

	if h.session.Exists(r, "user") {
		user, ok := h.session.Get(r, "user").(nakama.User)
		out.IsLoggedIn = ok
		out.User = user
	}

	return out
}
