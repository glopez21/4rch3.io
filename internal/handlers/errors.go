package handlers

import (
	"encoding/json"
	"net/http"
)

type apiError struct {
	Error   string `json:"error"`
	Status  int    `json:"-"`
}

func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiError{Error: message})
}
