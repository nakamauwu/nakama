package web

import (
	"net/http"

	"github.com/nakamauwu/nakama/web/templates"
)

// showWelcome handles GET /
func (h *Handler) showWelcome(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, templates.Welcome(), http.StatusOK)
}
