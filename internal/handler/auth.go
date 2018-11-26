package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nicolasparada/nakama/internal/service"
)

type loginInput struct {
	Email string
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.Login(r.Context(), in.Email)
	if err == service.ErrInvalidEmail {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, out, http.StatusOK)
}
