package web

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"syscall"

	"github.com/a-h/templ"
	"github.com/nakamauwu/nakama/web/templates"
	"github.com/nicolasparada/go-errs"
)

func (h *Handler) renderErr(w http.ResponseWriter, r *http.Request, err error) {
	code := err2code(err)
	if code == http.StatusInternalServerError {
		h.Logger.Error("render", "err", err)
		h.render(w, r, templates.Error(errors.New("internal server error")), http.StatusInternalServerError)
		return
	}

	h.Logger.Warn("render", "err", err)
	h.render(w, r, templates.Error(err), code)
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, comp templ.Component, statusCode int) {
	var buff bytes.Buffer
	if err := comp.Render(r.Context(), &buff); err != nil {
		h.renderErr(w, r, fmt.Errorf("render template: %w", err))
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
