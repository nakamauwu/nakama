package web

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
)

var loginTmpl = parseTmpl("login.tmpl")

type loginData struct {
	Form url.Values
	Err  error
}

func (h *Handler) renderLogin(w http.ResponseWriter, data loginData, statusCode int) {
	h.renderTmpl(w, loginTmpl, data, statusCode)
}

func (h *Handler) showLogin(w http.ResponseWriter, r *http.Request) {
	h.renderLogin(w, loginData{}, http.StatusOK)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	input := nakama.LoginInput{
		Email:    r.PostFormValue("email"),
		Username: formPtr(r.PostForm, "username"),
	}
	user, err := h.Service.Login(ctx, input)
	if errors.Is(err, nakama.ErrUserNotFound) || errors.Is(err, nakama.ErrUsernameTaken) {
		h.renderLogin(w, loginData{
			Form: r.PostForm,
			Err:  err,
		}, http.StatusBadRequest)
		return
	}
	if err != nil {
		h.Logger.Printf("could not login: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.session.Put(r, "user", user)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.session.Remove(r, "user")
	http.Redirect(w, r, "/", http.StatusFound)
}

func formPtr(form url.Values, key string) *string {
	if !form.Has(key) {
		return nil
	}

	s := form.Get(key)
	return &s
}
