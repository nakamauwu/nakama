package web

import (
	"errors"
	"net/http"

	"github.com/nicolasparada/go-errs/httperrs"
)

var errInternalServerError = errors.New("internal server error")

var errorPage = parsePage("error-page.tmpl")

type errData struct {
	Session
	Err error
}

func (h *Handler) renderErr(w http.ResponseWriter, r *http.Request, err error) {
	h.render(w, errorPage, errData{
		Session: h.sessionFromReq(r),
		Err:     maskErr(err),
	}, httperrs.Code(err))
}

// maskErr returns an internal server error
// if the error is not a sentinel error from nakama package.
// Use this to not leak internal error details to the user.
func maskErr(err error) error {
	if httperrs.IsInternalServerError(err) {
		return errInternalServerError
	}
	return err
}
