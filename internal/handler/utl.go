package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func respond(w http.ResponseWriter, v interface{}, statusCode int) {
	b, err := json.Marshal(v)
	if err != nil {
		respondError(w, fmt.Errorf("could not marshal response: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(b)
}

func respondError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
