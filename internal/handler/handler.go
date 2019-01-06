package handler

import (
	"net/http"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/internal/service"
)

type handler struct {
	*service.Service
}

// New makes use of the service to provide an http.Handler with predefined routing.
func New(s *service.Service) http.Handler {
	h := &handler{s}

	api := way.NewRouter()
	api.HandleFunc("POST", "/login", h.login)
	api.HandleFunc("GET", "/auth_user", h.authUser)
	api.HandleFunc("POST", "/users", h.createUser)
	api.HandleFunc("GET", "/users", h.users)
	api.HandleFunc("GET", "/users/:username", h.user)
	api.HandleFunc("PUT", "/auth_user/avatar", h.updateAvatar)
	api.HandleFunc("POST", "/users/:username/toggle_follow", h.toggleFollow)
	api.HandleFunc("GET", "/users/:username/followers", h.followers)
	api.HandleFunc("GET", "/users/:username/followees", h.followees)
	api.HandleFunc("POST", "/posts", h.createPost)
	api.HandleFunc("POST", "/posts/:post_id/toggle_like", h.togglePostLike)

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", h.withAuth(api)))

	return r
}
