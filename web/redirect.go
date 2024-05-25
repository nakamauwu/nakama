package web

import "net/http"

const sessKeyFlashError = "flash_error"

func (h *Handler) flashError(r *http.Request, err error) {
	h.sess.Put(r.Context(), sessKeyFlashError, err.Error())
}

func (h *Handler) goBackWithError(w http.ResponseWriter, r *http.Request, err error) {
	h.flashError(r, err)
	goBack(w, r)
}

func goBack(w http.ResponseWriter, r *http.Request) {
	// SeeOther so that refreshing the result page doesn't re-trigger the operation
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}
