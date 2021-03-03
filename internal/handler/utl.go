package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var errStreamingUnsupported = errors.New("streaming unsupported")

func respond(w http.ResponseWriter, v interface{}, statusCode int) {
	b, err := json.Marshal(v)
	if err != nil {
		respondErr(w, fmt.Errorf("could not marshal response: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(b)
}

func respondErr(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func writeSSE(w io.Writer, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("could not marshal response: %v\n", err)
		fmt.Fprintf(w, "event: error\ndata: %v\n\n", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", b)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	targetStr := r.URL.Query().Get("target")
	if targetStr == "" {
		respondErr(w, errors.New("invalid target URL"))
		return
	}

	target, err := url.Parse(targetStr)
	if err != nil || !target.IsAbs() {
		respondErr(w, errors.New("invalid target URL"))
		return
	}

	director := func(newr *http.Request) {
		newr.Host = r.URL.Host
		newr.RequestURI = target.String()
		newr.URL = target

		// taken from httputil.NewSingleHostReverseProxy
		if _, ok := newr.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			newr.Header.Set("User-Agent", "")
		}
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

func withCacheControl(d time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", int64(d.Seconds())))
			h(w, r)
		}
	}
}
