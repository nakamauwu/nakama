package web

import (
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
)

var homeTmpl = parseTmpl("home.tmpl")

type homeData struct {
	Session
	CreatePostErr  error
	CreatePostForm url.Values
	Timeline       []nakama.HomeTimelineRow
	Posts          []nakama.PostsRow
}

func (h *Handler) renderHome(w http.ResponseWriter, data homeData, statusCode int) {
	h.renderTmpl(w, homeTmpl, data, statusCode)
}

// showHome handles GET /.
func (h *Handler) showHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tt, err := h.Service.HomeTimeline(ctx)
	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	// TODO: show either timeline or posts based whether the user is logged in.
	// pp, err := h.Service.Posts(ctx, nakama.PostsInput{})
	// if err != nil {
	// 	h.log(err)
	// 	h.renderErr(w, r, err)
	// 	return
	// }

	h.renderHome(w, homeData{
		Session:        h.sessionFromReq(r),
		CreatePostErr:  h.popErr(r, "create_post_err"),
		CreatePostForm: h.popForm(r, "create_post_form"),
		Timeline:       tt,
		// Posts:          pp,
	}, http.StatusOK)
}
