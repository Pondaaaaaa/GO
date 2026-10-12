package utils

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

func WriteErr(w http.ResponseWriter, status int, message string) error {
	return WriteJSON(w, status, struct {
		Error string `json:"error"`
	}{Error: message})
}
