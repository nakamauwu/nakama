package web

import (
	"net/http"

	"github.com/nakamauwu/nakama/types"
)

// showPosts handles GET /
func (h *Handler) showPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	posts, err := h.Service.Posts(ctx, types.ListPosts{})
	if err != nil {
		h.renderErr(w, r, err)
		return
	}

	h.render(w, r, "posts.tmpl", map[string]any{
		"Session": h.sessionData(r),
		"Posts":   posts,
	}, http.StatusOK)
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var in types.CreatePost
	if err := h.decode(r, &in); err != nil {
		h.goBackWithError(w, r, err)
		return
	}

	ctx := r.Context()
	_, err := h.Service.CreatePost(ctx, in)
	if err != nil {
		h.goBackWithError(w, r, err)
		return
	}

	goBack(w, r)
}
