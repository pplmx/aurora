package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenHandler_Create_InvalidJSON(t *testing.T) {
	handler := NewTokenHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Create_EmptyRequest(t *testing.T) {
	handler := NewTokenHandler(nil)

	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTokenHandler_Mint_InvalidJSON(t *testing.T) {
	handler := NewTokenHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/mint", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Mint(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Mint_EmptyRequest(t *testing.T) {
	handler := NewTokenHandler(nil)

	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/mint", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Mint(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTokenHandler_Transfer_InvalidJSON(t *testing.T) {
	handler := NewTokenHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/transfer", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Transfer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Transfer_EmptyRequest(t *testing.T) {
	handler := NewTokenHandler(nil)

	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/transfer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Transfer(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTokenHandler_Burn_InvalidJSON(t *testing.T) {
	handler := NewTokenHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/burn", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Burn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Burn_EmptyRequest(t *testing.T) {
	handler := NewTokenHandler(nil)

	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/burn", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Burn(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTokenHandler_Balance_MissingTokenID(t *testing.T) {
	handler := NewTokenHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/balance?owner=test", nil)
	rr := httptest.NewRecorder()

	handler.Balance(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Balance_MissingOwner(t *testing.T) {
	handler := NewTokenHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/balance?token_id=test", nil)
	rr := httptest.NewRecorder()

	handler.Balance(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Balance_BothMissing(t *testing.T) {
	handler := NewTokenHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/balance", nil)
	rr := httptest.NewRecorder()

	handler.Balance(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_ResponseContentType(t *testing.T) {
	handler := NewTokenHandler(nil)

	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	require.NotEmpty(t, rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
}