package web

import "net/http"

// showWelcome handles GET /
func (h *Handler) showWelcome(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "welcome.tmpl", map[string]any{
		"Session": h.sessionData(r),
	}, http.StatusOK)
}
