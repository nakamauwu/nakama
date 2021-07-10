package http

import (
	"encoding/json"
	"net/http"
)

type pushLogReqBody struct {
	Error string `json:"error"`
}

func (h *handler) pushLog(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("origin") != h.origin.String() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	defer r.Body.Close()

	var reqBody pushLogReqBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	if reqBody.Error == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_ = h.logger.Log("subcomponent", "client", "err", reqBody.Error)
	w.WriteHeader(http.StatusNoContent)
}
