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

func TestNFTHandler_Mint_InvalidJSON(t *testing.T) {
	handler := NewNFTHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Mint(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_Mint_EmptyRequest(t *testing.T) {
	handler := NewNFTHandler(nil)

	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Mint(rr, req)

	// Empty request triggers domain validation errors (name required,
	// invalid creator base64), now properly classified as 400.
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_Transfer_InvalidJSON(t *testing.T) {
	handler := NewNFTHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/transfer", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Transfer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_Routes(t *testing.T) {
	handler := NewNFTHandler(nil)
	assert.NotNil(t, handler)
}

func TestNFTHandler_Mint_ResponseContentType(t *testing.T) {
	handler := NewNFTHandler(nil)

	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Mint(rr, req)

	require.NotEmpty(t, rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
}

func TestNFTHandler_Burn_InvalidJSON(t *testing.T) {
	handler := NewNFTHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/burn", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Burn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_List_EmptyOwner(t *testing.T) {
	handler := NewNFTHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/list", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	// Empty owner -> 400 Bad Request (validated before service call)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_List_InvalidOwner(t *testing.T) {
	handler := NewNFTHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/list?owner=!!!invalid-base64!!!", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	// Invalid base64 owner -> unclassified error -> 500
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
