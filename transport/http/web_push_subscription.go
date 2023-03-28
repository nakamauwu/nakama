package http

import (
	"encoding/json"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
)

func (h *handler) addWebPushSubscription(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var sub webpush.Subscription
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
