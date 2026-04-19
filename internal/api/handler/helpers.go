package handler

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(w http.ResponseWriter, message string, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	})
}

func writeInternalError(w http.ResponseWriter) {
	writeError(w, "internal server error", "INTERNAL_ERROR", http.StatusInternalServerError)
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeError(w, message, "INVALID_REQUEST", http.StatusBadRequest)
}
