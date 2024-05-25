package web

import "net/http"

// showPosts handles GET /
func (h *Handler) showPosts(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "posts.tmpl", map[string]any{
		"Session": h.sessionData(r),
	}, http.StatusOK)
}
