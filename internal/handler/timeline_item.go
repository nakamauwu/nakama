package handler

import (
	"net/http"
	"strconv"

	"github.com/nicolasparada/nakama/internal/service"
)

func (h *handler) timeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	tt, err := h.Timeline(ctx, last, before)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, tt, http.StatusOK)
}
