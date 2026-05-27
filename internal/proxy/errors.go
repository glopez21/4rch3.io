package proxy

import (
	"encoding/json"
	"net/http"
)

type proxyError struct {
	code    int
	message string
}

func (e *proxyError) Error() string { return e.message }

var (
	errUnauthorized       = &proxyError{http.StatusUnauthorized, "unauthorized"}
	errServiceNotSpecified = &proxyError{http.StatusBadRequest, "service not specified"}
	errServiceNotFound    = &proxyError{http.StatusNotFound, "service not found"}
	errInvalidTarget      = &proxyError{http.StatusInternalServerError, "invalid target"}
	errInternal           = &proxyError{http.StatusInternalServerError, "internal error"}
)

type errorResponse struct {
	Error   string `json:"error"`
	Service string `json:"service,omitempty"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}
