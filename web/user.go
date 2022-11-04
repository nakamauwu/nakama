package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
	"golang.org/x/sync/errgroup"
)

var userPageTmpl = parseTmpl("user-page.tmpl")

type userData struct {
	Session
	User          nakama.User
	Posts         []nakama.Post
	UserFollowErr error
}

func (h *Handler) renderUser(w http.ResponseWriter, data userData, statusCode int) {
	h.renderTmpl(w, userPageTmpl, data, statusCode)
}

// showUser handles GET /@{username}.
func (h *Handler) showUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := mux.URLParam(ctx, "username")

	var user nakama.User
	var posts []nakama.Post

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		user, err = h.Service.User(gctx, username)
		return err
	})

	g.Go(func() error {
		var err error
		posts, err = h.Service.Posts(gctx, nakama.PostsInput{Username: username})
		return err
	})

	if err := g.Wait(); err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderUser(w, userData{
		Session:       h.sessionFromReq(r),
		User:          user,
		Posts:         posts,
		UserFollowErr: h.popErr(r, "user_follow_err"),
	}, http.StatusOK)
}
