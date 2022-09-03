package web

import (
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
)

var homePageTmpl = parseTmpl("home-page.tmpl")

type homeData struct {
	Session
	CreatePostErr  error
	CreatePostForm url.Values
	Posts          any
}

func (h *Handler) renderHome(w http.ResponseWriter, data homeData, statusCode int) {
	h.renderTmpl(w, homePageTmpl, data, statusCode)
}

// showHome handles GET /.
func (h *Handler) showHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var posts any
	var err error

	if _, ok := nakama.UserFromContext(ctx); ok {
		posts, err = h.Service.HomeTimeline(ctx)
	} else {
		posts, err = h.Service.Posts(ctx, nakama.PostsInput{})
	}

	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderHome(w, homeData{
		Session:        h.sessionFromReq(r),
		CreatePostErr:  h.popErr(r, "create_post_err"),
		CreatePostForm: h.popForm(r, "create_post_form"),
		Posts:          posts,
	}, http.StatusOK)
}
