package web

import (
	"net/http"

	"github.com/nakamauwu/nakama/types"
	"github.com/nakamauwu/nakama/web/templates"
)

// showPosts handles GET /
func (h *Handler) showPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in types.ListPosts
	if err := h.decodeQuery(r, &in); err != nil {
		h.renderErr(w, r, err)
		return
	}

	posts, err := h.Service.Posts(ctx, in)
	if err != nil {
		h.renderErr(w, r, err)
		return
	}

	h.render(w, r, templates.Posts(templates.PostsData{
		Posts: posts,
	}), http.StatusOK)
}

// createPost handles POST /posts
func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var in types.CreatePost
	if err := h.decodePostForm(r, &in); err != nil {
		h.goBackWithError(w, r, err)
		return
	}

	ctx := r.Context()
	_, err := h.Service.CreatePost(ctx, in)
	if err != nil {
		h.goBackWithError(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
