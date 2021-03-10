package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/internal/storage"
)

func (h *handler) avatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := way.Param(ctx, "name")

	f, err := h.store.Open(ctx, name)
	if err == storage.ErrNotFound {
		respond(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	w.Header().Set("Etag", f.ETag)
	w.Header().Set("Last-Modified", f.LastModified.Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, f)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("could not write down avatar: %v\n", err)
	}
}
