package web

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-errs/httperrs"
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
		h.renderLogin(w, loginData{Err: errors.New("bad request")}, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	input := nakama.LoginInput{
		Email:    r.PostFormValue("email"),
		Username: formPtr(r.PostForm, "username"),
	}
	user, err := h.Service.Login(ctx, input)
	if err != nil {
		h.log(err)
		h.renderLogin(w, loginData{
			Form: r.PostForm,
			Err:  maskErr(err),
		}, httperrs.Code(err))
		return
	}

	h.session.Put(r, "user", user)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) log(err error) {
	if httperrs.IsInternalServerError(err) {
		_ = h.Logger.Output(2, err.Error())
	}
}

func maskErr(err error) error {
	if httperrs.IsInternalServerError(err) {
		return errors.New("internal server error")
	}
	return err
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

func (h *Handler) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.session.Exists(r, "user") {
			next.ServeHTTP(w, r)
			return
		}

		usr, ok := h.session.Get(r, "user").(nakama.User)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = nakama.ContextWithUser(ctx, usr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
