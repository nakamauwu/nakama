package web

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
)

type Session struct {
	IsLoggedIn bool
	User       nakama.UserIdentity
}

func (h *Handler) sessionFromReq(r *http.Request) Session {
	ctx := r.Context()
	usr, ok := nakama.UserFromContext(ctx)
	return Session{
		IsLoggedIn: ok,
		User:       usr,
	}
}

func (h *Handler) getUserIdentity(r *http.Request, key string) nakama.UserIdentity {
	v, _ := h.session.Get(r, key).(nakama.UserIdentity)
	return v
}

func (h *Handler) putErr(r *http.Request, key string, err error) {
	h.session.Put(r, key, maskErr(err).Error())
}

func (h *Handler) popErr(r *http.Request, key string) error {
	s := h.session.PopString(r, key)
	if s != "" {
		return errors.New(s)
	}
	return nil
}

func (h *Handler) popForm(r *http.Request, key string) url.Values {
	v, _ := h.session.Pop(r, key).(url.Values)
	return v
}
