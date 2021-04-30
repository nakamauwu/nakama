package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/nicolasparada/nakama"
	"github.com/nicolasparada/nakama/storage"
)

var (
	errBadRequest           = errors.New("bad request")
	errStreamingUnsupported = errors.New("streaming unsupported")
	errTeaPot               = errors.New("i am a teapot")
	errInvalidTargetURL     = nakama.InvalidArgumentError("invalid target URL")
)

func respond(w http.ResponseWriter, v interface{}, statusCode int) {
	b, err := json.Marshal(v)
	if err != nil {
		respondErr(w, fmt.Errorf("could not json marshal http response body: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err = w.Write(b)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("could not write down http response: %v\n", err)
	}
}

func respondErr(w http.ResponseWriter, err error) {
	statusCode := err2code(err)
	if statusCode == http.StatusInternalServerError {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Error(w, err.Error(), statusCode)
}

func err2code(err error) int {
	if err == nil {
		return http.StatusOK
	}

	switch {
	case err == errBadRequest ||
		err == errWebAuthnTimeout:
		return http.StatusBadRequest
	case err == errStreamingUnsupported:
		return http.StatusExpectationFailed
	case err == errTeaPot:
		return http.StatusTeapot
	case errors.Is(err, nakama.ErrInvalidArgument):
		return http.StatusUnprocessableEntity
	case errors.Is(err, nakama.ErrNotFound) ||
		errors.Is(err, storage.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, nakama.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, nakama.ErrPermissionDenied):
		return http.StatusForbidden
	case err == nakama.ErrUnauthenticated || errors.Is(err, nakama.ErrUnauthenticated):
		return http.StatusUnauthorized
	case errors.Is(err, nakama.ErrUnimplemented):
		return http.StatusNotImplemented
	case errors.Is(err, nakama.ErrGone):
		return http.StatusGone
	}

	return http.StatusInternalServerError
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
		respondErr(w, errInvalidTargetURL)
		return
	}

	target, err := url.Parse(targetStr)
	if err != nil || !target.IsAbs() {
		respondErr(w, errInvalidTargetURL)
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