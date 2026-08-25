package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	domainnft "github.com/pplmx/aurora/internal/domain/nft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNFTHandler_Mint_InvalidJSON(t *testing.T) {
	handler := NewNFTHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Mint(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_Mint_EmptyRequest(t *testing.T) {
	handler := NewNFTHandler(nil, nil)

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
	handler := NewNFTHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/transfer", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Transfer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_Routes(t *testing.T) {
	handler := NewNFTHandler(nil, nil)
	assert.NotNil(t, handler)
}

func TestNFTHandler_History_ReturnsMintOperation(t *testing.T) {
	repo := domainnft.NewInmemRepo()
	handler := NewNFTHandler(repo, nil)

	creator := make([]byte, 32)
	for i := range creator {
		creator[i] = byte(i)
	}
	mintBody, _ := json.Marshal(map[string]string{
		"name":    "MyNFT",
		"creator": base64.StdEncoding.EncodeToString(creator),
	})
	mintReq := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewBuffer(mintBody))
	mintReq.Header.Set("Content-Type", "application/json")
	mintRR := httptest.NewRecorder()
	handler.Mint(mintRR, mintReq)
	require.Equal(t, http.StatusOK, mintRR.Code)

	var minted struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(mintRR.Body.Bytes(), &minted))
	require.NotEmpty(t, minted.ID)

	// Look up the operation trail for the minted NFT (v1.35).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/"+minted.ID+"/history", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", minted.ID)
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var ops []json.RawMessage
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ops))
	assert.True(t, len(ops) >= 1, "expected at least the mint operation, got %d", len(ops))
	require.Contains(t, rr.Body.String(), `"type":"mint"`)
}

func TestNFTHandler_Mint_ResponseContentType(t *testing.T) {
	handler := NewNFTHandler(nil, nil)

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
	handler := NewNFTHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/burn", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Burn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_List_EmptyOwner(t *testing.T) {
	handler := NewNFTHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/list", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	// Empty owner -> 400 Bad Request (validated before service call)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestNFTHandler_List_InvalidOwner(t *testing.T) {
	handler := NewNFTHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/list?owner=!!!invalid-base64!!!", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	// Invalid base64 owner -> client error -> 400 INVALID_BASE64 (TASK-095,
	// ISS-089; previously an unclassified 500).
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_BASE64", resp.Code)
}
