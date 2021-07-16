package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/nicolasparada/nakama"
	"github.com/nicolasparada/nakama/storage"
)

const proxyCacheControl = time.Hour * 24 * 14

var (
	errBadRequest           = errors.New("bad request")
	errStreamingUnsupported = errors.New("streaming unsupported")
	errTeaPot               = errors.New("i am a teapot")
	errInvalidTargetURL     = nakama.InvalidArgumentError("invalid target URL")
	errWebAuthnTimeout      = errors.New("webAuthn timeout")
	errOauthTimeout         = errors.New("oauth timeout")
	errEmailNotVerified     = errors.New("email not verified")
	errEmailNotProvided     = errors.New("email not provided")
	errServiceUnavailable   = errors.New("service unavailable")
)

type paginatedRespBody struct {
	Items       interface{} `json:"items"`
	StartCursor *string     `json:"startCursor"`
	EndCursor   *string     `json:"endCursor"`
}

func (h *handler) respond(w http.ResponseWriter, v interface{}, statusCode int) {
	b, err := json.Marshal(v)
	if err != nil {
		h.respondErr(w, fmt.Errorf("could not json marshal http response body: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err = w.Write(b)
	if err != nil && !errors.Is(err, syscall.EPIPE) && !errors.Is(err, context.Canceled) {
		_ = h.logger.Log("err", fmt.Errorf("could not write down http response: %w", err))
	}
}

func (h *handler) respondErr(w http.ResponseWriter, err error) {
	statusCode := err2code(err)
	if statusCode == http.StatusInternalServerError {
		if !errors.Is(err, context.Canceled) {
			_ = h.logger.Log("err", err)
		}
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
		err == errWebAuthnTimeout ||
		err == errOauthTimeout ||
		err == errEmailNotVerified ||
		err == errEmailNotProvided:
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
	case err == errServiceUnavailable:
		return http.StatusServiceUnavailable
	}

	return http.StatusInternalServerError
}

func (h *handler) writeSSE(w io.Writer, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		_ = h.logger.Log("err", fmt.Errorf("could not json marshal sse data: %w", err))
		fmt.Fprintf(w, "event: error\ndata: %v\n\n", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", b)
}

func (h *handler) proxy(w http.ResponseWriter, r *http.Request) {
	targetStr := r.URL.Query().Get("target")
	if targetStr == "" {
		h.respondErr(w, errInvalidTargetURL)
		return
	}

	target, err := url.Parse(targetStr)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") {
		h.respondErr(w, errInvalidTargetURL)
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
	proxy := &httputil.ReverseProxy{
		Director: director,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusBadGateway)
		},
	}
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

func emptyStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func redirectWithHashFragment(w http.ResponseWriter, r *http.Request, uri *url.URL, data url.Values, statusCode int) {
	// Using query string instead of hash fragment because golang's url.URL#RawFragment is a no-op.
	// We set the RawQuery instead, and then string replace the "?" symbol by "#".
	uri.RawQuery = data.Encode()
	location := uri.String()
	location = strings.Replace(location, "?", "#", 1)
	http.Redirect(w, r, location, statusCode)
}
