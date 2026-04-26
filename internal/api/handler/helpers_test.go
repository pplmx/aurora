package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteInternalError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeInternalError(rr)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "internal server error", resp.Error)
	assert.Equal(t, "INTERNAL_ERROR", resp.Code)
}

func TestWriteBadRequest(t *testing.T) {
	rr := httptest.NewRecorder()

	writeBadRequest(rr, "test message")

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "test message", resp.Error)
	assert.Equal(t, "INVALID_REQUEST", resp.Code)
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeError(rr, "custom error", "CUSTOM_CODE", http.StatusForbidden)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "custom error", resp.Error)
	assert.Equal(t, "CUSTOM_CODE", resp.Code)
}
