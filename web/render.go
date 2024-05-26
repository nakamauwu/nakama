package web

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"syscall"

	"github.com/nakamauwu/nakama/auth"
	"github.com/nicolasparada/go-errs"
	tmplrenderer "github.com/nicolasparada/go-tmpl-renderer"
	"mvdan.cc/xurls/v2"
)

//go:embed templates/includes/*.tmpl templates/*.tmpl
var templatesFS embed.FS

func (h *Handler) sessionData(r *http.Request) map[string]any {
	ctx := r.Context()
	user, isLoggedIn := auth.UserFromContext(ctx)
	return map[string]any{
		"User":       user,
		"IsLoggedIn": isLoggedIn,
		"CurrentURL": r.URL.Path,
		"FlashError": h.sess.PopString(ctx, sessKeyFlashError),
	}
}

func newRenderer() *tmplrenderer.Renderer {
	return &tmplrenderer.Renderer{
		FS:             templatesFS,
		BaseDir:        "templates",
		IncludePatters: []string{"includes/*.tmpl"},
		FuncMap: template.FuncMap{
			"linkify": linkify,
		},
	}
}

func (h *Handler) renderErr(w http.ResponseWriter, r *http.Request, err error) {
	code := err2code(err)
	if code == http.StatusInternalServerError {
		h.Logger.Error("render", "err", err)
		h.render(w, r, "error.tmpl", map[string]any{
			"Session": h.sessionData(r),
			"Error":   "internal server error",
		}, code)
		return
	}

	h.Logger.Warn("render", "err", err)
	h.render(w, r, "error.tmpl", map[string]any{
		"Session": h.sessionData(r),
		"Error":   err,
	}, code)
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, tmplName string, data map[string]any, statusCode int) {
	var buff bytes.Buffer
	if err := h.renderer.Render(&buff, tmplName, data); err != nil {
		h.renderErr(w, r, fmt.Errorf("render %q: %w", tmplName, err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err := buff.WriteTo(w)
	if err != nil && !errors.Is(err, syscall.EPIPE) {
		h.Logger.Error("write response", "err", err)
	}
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	h.renderErr(w, r, errs.NotFoundError("page not found"))
}

var reURL = xurls.Relaxed()

func linkify(s string) template.HTML {
	s = template.HTMLEscapeString(s)
	s = reURL.ReplaceAllString(s,
		`<a href="$0" target="_blank" rel="noopener noreferrer">$0</a>`,
	)
	return template.HTML(s)
}
