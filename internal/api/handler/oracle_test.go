package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOracleHandler_Fetch_InvalidJSON(t *testing.T) {
	handler := NewOracleHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/fetch", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Fetch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOracleHandler_Routes(t *testing.T) {
	handler := NewOracleHandler(nil)
	assert.NotNil(t, handler)
}