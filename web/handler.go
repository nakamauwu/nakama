package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/nakamauwu/nakama/auth"
	"github.com/nakamauwu/nakama/oauth"
	"github.com/nakamauwu/nakama/service"
	tmplrenderer "github.com/nicolasparada/go-tmpl-renderer"
)

type Handler struct {
	Logger       *slog.Logger
	SessionStore scs.Store
	Service      *service.Service
	Providers    []oauth.Provider

	renderer *tmplrenderer.Renderer
	sess     *scs.SessionManager

	once    sync.Once
	handler http.Handler
}

func (h *Handler) init() {
	h.renderer = newRenderer()
	h.sess = scs.New()
	h.sess.Store = h.SessionStore
	h.sess.Lifetime = time.Hour * 24 * 14 // 2 weeks
	// h.sess.Cookie.Secure = true

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserFromContext(r.Context()); ok {
			h.showPosts(w, r)
		} else {
			h.showWelcome(w, r)
		}
	})

	for _, provider := range h.Providers {
		mux.HandleFunc(fmt.Sprintf("GET /oauth2/%s/redirect", provider.Name()), h.providerRedirect(provider))
		mux.HandleFunc(fmt.Sprintf("GET /oauth2/%s/callback", provider.Name()), h.providerCallback(provider))
	}

	mux.HandleFunc("GET /login", h.showLogin)
	mux.HandleFunc("POST /logout", h.logout)
	mux.Handle("GET /static/", h.staticHandler())
	mux.HandleFunc("GET /", h.notFound)

	// middlwares are registered in reverse order
	h.handler = mux
	h.handler = h.withUser(h.handler)
	h.handler = h.sess.LoadAndSave(h.handler)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(h.init)
	h.handler.ServeHTTP(w, r)
}
