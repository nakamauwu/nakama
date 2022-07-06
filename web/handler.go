package web

import (
	"encoding/gob"
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
	once       sync.Once
	handler    http.Handler
	session    *sessions.Session
}

func (h *Handler) init() {
	r := mux.NewRouter()

	r.Handle("/", mux.MethodHandler{
		http.MethodGet: h.showHome,
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
		http.MethodGet: h.post,
	})

	r.Handle("/*", mux.MethodHandler{
		http.MethodGet: h.static(),
	})

	gob.Register(nakama.User{})
	gob.Register(url.Values{})
	h.session = sessions.New(h.SessionKey)

	h.handler = r
	h.handler = h.withUser(h.handler)
	h.handler = h.session.Enable(h.handler)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(h.init)
	h.handler.ServeHTTP(w, r)
}

func (h *Handler) log(err error) {
	if httperrs.IsInternalServerError(err) {
		_ = h.Logger.Output(2, err.Error())
	}
}

func formPtr(form url.Values, key string) *string {
	if !form.Has(key) {
		return nil
	}

	s := form.Get(key)
	return &s
}
