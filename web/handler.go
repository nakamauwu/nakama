package web

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/golangcollege/sessions"
	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-errs/httperrs"
	"github.com/nicolasparada/go-mux"
)

type Handler struct {
	Logger     *log.Logger
	Service    *nakama.Service
	SessionKey []byte

	session *sessions.Session

	once    sync.Once
	handler http.Handler
}

func (h *Handler) init() {
	r := mux.NewRouter()

	r.Handle("/", mux.MethodHandler{
		http.MethodGet: h.showPosts,
	})

	r.Handle("/login", mux.MethodHandler{
		http.MethodGet:  h.showLogin,
		http.MethodPost: h.login,
	})

	r.Handle("/logout", mux.MethodHandler{
		http.MethodPost: h.logout,
	})

	r.Handle("/posts", mux.MethodHandler{
		http.MethodPost: h.createPost,
	})

	r.Handle("/p/{postID}", mux.MethodHandler{
		http.MethodGet: h.showPost,
	})

	r.Handle("/p/{postID}/reactions", mux.MethodHandler{
		http.MethodPost: h.addPostReaction,
	})

	r.Handle("/comments", mux.MethodHandler{
		http.MethodPost: h.createComment,
	})

	r.Handle("/@{username}", mux.MethodHandler{
		http.MethodGet: h.showUser,
	})

	r.Handle("/user-follows", mux.MethodHandler{
		http.MethodPost:   h.followUser,
		http.MethodDelete: h.unfollowUser,
	})

	r.Handle("/settings", mux.MethodHandler{
		http.MethodGet: h.showSettings,
	})

	r.Handle("/avatar", mux.MethodHandler{
		http.MethodPut: h.updateAvatar,
	})

	r.Handle("/user", mux.MethodHandler{
		http.MethodPatch: h.updateUser,
	})

	r.Handle("/search", mux.MethodHandler{
		http.MethodGet: h.showSearch,
	})

	r.Handle("/*", mux.MethodHandler{
		http.MethodGet: h.staticHandler(),
	})

	// register types used on sessions.
	gob.Register(nakama.UserIdentity{})
	gob.Register(url.Values{})

	h.session = sessions.New(h.SessionKey)

	h.handler = r
	h.handler = h.withUser(h.handler)
	h.handler = h.session.Enable(h.handler)
	h.handler = withMethodOverride(h.handler)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(h.init)
	h.handler.ServeHTTP(w, r)
}

// log an error only if it's an internal error.
func (h *Handler) log(err error) {
	if httperrs.IsInternalServerError(err) && !errors.Is(err, context.Canceled) {
		_ = h.Logger.Output(2, err.Error())
	}
}

// formPtr utility to get a *string from a form.
func formPtr(form url.Values, key string) *string {
	if !form.Has(key) {
		return nil
	}

	s := form.Get(key)
	return &s
}

func withMethodOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only act on POST requests.
		if r.Method == "POST" {

			// Look in the request body and headers for a spoofed method.
			// Prefer the value in the request body if they conflict.
			method := r.PostFormValue("_method")

			// Check that the spoofed method is a valid HTTP method and
			// update the request object accordingly.
			if method == "PUT" || method == "PATCH" || method == "DELETE" {
				r.Method = method
			}
		}

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}
