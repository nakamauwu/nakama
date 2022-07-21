package web

import (
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-errs/httperrs"
)

const sessionKeyUser = "user"

var loginTmpl = parseTmpl("login.tmpl")

type loginData struct {
	Session
	Form url.Values
	Err  error
}

func (h *Handler) renderLogin(w http.ResponseWriter, data loginData, statusCode int) {
	h.renderTmpl(w, loginTmpl, data, statusCode)
}

// showLogin handles GET /login.
func (h *Handler) showLogin(w http.ResponseWriter, r *http.Request) {
	h.renderLogin(w, loginData{
		Session: h.sessionFromReq(r),
	}, http.StatusOK)
}

// login handles POST /login.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		h.renderLogin(w, loginData{
			Session: h.sessionFromReq(r),
			Err:     err,
		}, http.StatusBadRequest)
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
			Session: h.sessionFromReq(r),
			Form:    r.PostForm,
			Err:     maskErr(err),
		}, httperrs.Code(err))
		return
	}

	h.session.Put(r, sessionKeyUser, user)
	http.Redirect(w, r, "/", http.StatusFound)
}

// logout handles POST /logout.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.session.Remove(r, sessionKeyUser)
	http.Redirect(w, r, "/", http.StatusFound)
}

// withUser middleware places the user from the session
// into the request's context.
// It continues to the next handler if user does not exists in session.
func (h *Handler) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usr, ok := h.session.Get(r, sessionKeyUser).(nakama.User)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = nakama.ContextWithUser(ctx, usr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
