package handler

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/internal/service"
)

func (h *handler) timeline(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/event-stream" {
		h.timelineItemSubscription(w, r)
		return
	}

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

func (h *handler) timelineItemSubscription(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondErr(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	tt, err := h.TimelineItemSubscription(ctx)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	header := w.Header()
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("Content-Type", "text/event-stream; charset=utf-8")

	for ti := range tt {
		writeSSE(w, ti)
		f.Flush()
	}
}

func (h *handler) deleteTimelineItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	timelineItemID, _ := strconv.ParseInt(way.Param(ctx, "timeline_item_id"), 10, 64)
	err := h.DeleteTimelineItem(ctx, timelineItemID)
	if err == service.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
