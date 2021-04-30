package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/matryer/way"
)

var epoch = time.Unix(0, 0).Format(time.RFC1123)

var noCacheHeaders = map[string]string{
	"Expires":         epoch,
	"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

type spaFileSystem struct {
	root http.FileSystem
}

func (fs *spaFileSystem) Open(name string) (http.File, error) {
	f, err := fs.root.Open(name)
	if os.IsNotExist(err) {
		return fs.root.Open("index.html")
	}
	return f, err
}

func withoutCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, v := range etagHeaders {
			r.Header.Del(v)
		}

		wh := w.Header()
		for k, v := range noCacheHeaders {
			wh.Set(k, v)
		}

		next.ServeHTTP(w, r)
	})
}

func (h *handler) avatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := way.Param(ctx, "name")

	f, err := h.store.Open(ctx, name)
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
