package http

import (
	"encoding/json"
	"net/http"
)

func (h *handler) addWebPushSubscription(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var sub json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	err := h.svc.AddWebPushSubscription(r.Context(), sub)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
