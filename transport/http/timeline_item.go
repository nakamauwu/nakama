package handler

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama"
)

func (h *handler) timeline(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/event-stream" {
		h.timelineItemStream(w, r)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before := q.Get("before")
	tt, err := h.svc.Timeline(ctx, last, before)
	if err == nakama.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, tt, http.StatusOK)
}

func (h *handler) timelineItemStream(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		respondErr(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	tt, err := h.svc.TimelineItemStream(ctx)
	if err == nakama.ErrUnauthenticated {
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

	select {
	case ti := <-tt:
		writeSSE(w, ti)
		f.Flush()
	case <-ctx.Done():
		return
	case <-h.ctx.Done():
		return
	}
}

func (h *handler) deleteTimelineItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	timelineItemID := way.Param(ctx, "timeline_item_id")
	err := h.svc.DeleteTimelineItem(ctx, timelineItemID)
	if err == nakama.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == nakama.ErrInvalidTimelineItemID {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
