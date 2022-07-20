package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
)

var userTmpl = parseTmpl("user.tmpl")

type userData struct {
	Session
	User          nakama.UserByUsernameRow
	Posts         []nakama.PostsRow
	UserFollowErr error
}

func (h *Handler) renderUser(w http.ResponseWriter, data userData, statusCode int) {
	h.renderTmpl(w, userTmpl, data, statusCode)
}

func (h *Handler) showUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := mux.URLParam(ctx, "username")

	// TODO: fetch both user and posts in parallel.
	usr, err := h.Service.UserByUsername(ctx, username)
	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	pp, err := h.Service.Posts(ctx, nakama.PostsInput{Username: username})
	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderUser(w, userData{
		Session:       h.sessionFromReq(r),
		User:          usr,
		Posts:         pp,
		UserFollowErr: h.popErr(r, "user_follow_err"),
	}, http.StatusOK)
}
