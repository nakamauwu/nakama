package handler

import (
	"net/http"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/internal/service"
)

type handler struct {
	*service.Service
}

// New creates an http.Handler with predefined routing.
// It makes use of the service to provide with an HTTP API.
func New(s *service.Service) http.Handler {
	h := &handler{s}

	api := way.NewRouter()
	api.HandleFunc("POST", "/login", h.login)
	api.HandleFunc("POST", "/users", h.createUser)

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", api))

	return r
}
