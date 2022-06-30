package web

import "net/http"

var homeTmpl = parseTmpl("home.tmpl")

type homeData struct {
	Session
}

func (h *Handler) renderHome(w http.ResponseWriter, data homeData, statusCode int) {
	h.renderTmpl(w, homeTmpl, data, statusCode)
}

func (h *Handler) showHome(w http.ResponseWriter, r *http.Request) {
	h.renderHome(w, homeData{
		Session: h.sessionFromReq(r),
	}, http.StatusOK)
}
