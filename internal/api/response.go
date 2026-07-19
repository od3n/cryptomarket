package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error string `json:"error"`
}

// PaginatedResponse wraps a list response with pagination metadata.
type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Count int         `json:"count"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
