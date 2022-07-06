package web

import (
	"net/http"

	"github.com/nicolasparada/go-errs/httperrs"
)

var errTmpl = parseTmpl("err.tmpl")

type errData struct {
	Session
	Err error
}

func (h *Handler) renderErr(w http.ResponseWriter, r *http.Request, err error) {
	h.renderTmpl(w, errTmpl, errData{
		Session: h.sessionFromReq(r),
		Err:     maskErr(err),
	}, httperrs.Code(err))
}
