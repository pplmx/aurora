package handler

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/domain/nft"
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
	repo := nft.NewInmemRepo()
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

// TestNFTHandler_History_AfterBurn locks ISS-359 on the REST surface: burn
// intentionally removes the NFT while retaining its operation trail, so
// GET /nft/{id}/history must keep returning that trail instead of 404ing.
func TestNFTHandler_History_AfterBurn(t *testing.T) {
	repo := nft.NewInmemRepo()
	handler := NewNFTHandler(repo, nil)

	owner, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	mintBody, err := json.Marshal(map[string]string{
		"name":    "Burned",
		"creator": base64.StdEncoding.EncodeToString(owner),
	})
	require.NoError(t, err)
	mintReq := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewReader(mintBody))
	mintReq.Header.Set("Content-Type", "application/json")
	mintRR := httptest.NewRecorder()
	handler.Mint(mintRR, mintReq)
	require.Equal(t, http.StatusOK, mintRR.Code)

	var minted struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(mintRR.Body.Bytes(), &minted))
	require.NotEmpty(t, minted.ID)

	burnBody, err := json.Marshal(map[string]string{
		"nft_id":      minted.ID,
		"owner":       base64.StdEncoding.EncodeToString(owner),
		"private_key": base64.StdEncoding.EncodeToString(privateKey),
	})
	require.NoError(t, err)
	burnReq := httptest.NewRequest(http.MethodPost, "/api/v1/nft/burn", bytes.NewReader(burnBody))
	burnReq.Header.Set("Content-Type", "application/json")
	burnRR := httptest.NewRecorder()
	handler.Burn(burnRR, burnReq)
	require.Equal(t, http.StatusOK, burnRR.Code)

	historyReq := httptest.NewRequest(http.MethodGet, "/api/v1/nft/"+minted.ID+"/history", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", minted.ID)
	historyReq = historyReq.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, routeCtx))
	historyRR := httptest.NewRecorder()
	handler.History(historyRR, historyReq)

	require.Equal(t, http.StatusOK, historyRR.Code)
	require.Contains(t, historyRR.Body.String(), `"type":"mint"`)
	require.Contains(t, historyRR.Body.String(), `"type":"burn"`)
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

func TestNFTHandler_RejectsZeroAddress(t *testing.T) {
	repo := nft.NewInmemRepo()
	handler := NewNFTHandler(repo, nil)
	zero := base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))

	mintBody, err := json.Marshal(map[string]string{
		"name":    "Zero Owner",
		"creator": zero,
	})
	require.NoError(t, err)
	mintReq := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewReader(mintBody))
	mintReq.Header.Set("Content-Type", "application/json")
	mintRR := httptest.NewRecorder()
	handler.Mint(mintRR, mintReq)
	require.Equal(t, http.StatusBadRequest, mintRR.Code)
	require.Contains(t, mintRR.Body.String(), `"code":"ZERO_ADDRESS"`)

	owner, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	validMintBody, err := json.Marshal(map[string]string{
		"name":    "Transferable",
		"creator": base64.StdEncoding.EncodeToString(owner),
	})
	require.NoError(t, err)
	validMintReq := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewReader(validMintBody))
	validMintReq.Header.Set("Content-Type", "application/json")
	validMintRR := httptest.NewRecorder()
	handler.Mint(validMintRR, validMintReq)
	require.Equal(t, http.StatusOK, validMintRR.Code)

	var minted struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(validMintRR.Body.Bytes(), &minted))

	transferBody, err := json.Marshal(map[string]string{
		"nft_id":      minted.ID,
		"from":        base64.StdEncoding.EncodeToString(owner),
		"to":          zero,
		"private_key": base64.StdEncoding.EncodeToString(privateKey),
	})
	require.NoError(t, err)
	transferReq := httptest.NewRequest(http.MethodPost, "/api/v1/nft/transfer", bytes.NewReader(transferBody))
	transferReq.Header.Set("Content-Type", "application/json")
	transferRR := httptest.NewRecorder()
	handler.Transfer(transferRR, transferReq)
	require.Equal(t, http.StatusBadRequest, transferRR.Code)
	require.Contains(t, transferRR.Body.String(), `"code":"ZERO_ADDRESS"`)
}

func TestNFTHandler_Burn_InvalidJSON(t *testing.T) {
	handler := NewNFTHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/burn", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Burn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// TestNFTHandler_List_WithoutOwnerListsEverything covers the case the old
// 400 blocked: a caller who did not mint the asset has no owner key to ask
// with, and requiring one made every NFT unreachable unless its id was already
// known.
func TestNFTHandler_List_WithoutOwnerListsEverything(t *testing.T) {
	repo := nft.NewInmemRepo()
	handler := NewNFTHandler(repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/list", nil)
	rr := httptest.NewRecorder()
	handler.List(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "[]\n", rr.Body.String(),
		"an empty collection must encode as [], not null and not a 400")
}

// TestNFTHandler_List_Paged locks the v1.79 bounded-paging fix (TASK-101,
// ISS-093): GET /nft/list?owner= must honor ?limit/?offset (default 20, cap
// 100) instead of returning an owner's entire collection unbounded.
func TestNFTHandler_List_Paged(t *testing.T) {
	handler := NewNFTHandler(nft.NewInmemRepo(), nil)

	// A 32-byte owner: the list use case rejects wrong-length owners as client
	// errors (TASK-112), and the in-memory matching is exact on the key bytes.
	ownerPub := bytes.Repeat([]byte{0x41}, 32)
	for i := 0; i < 5; i++ {
		err := handler.repo.SaveNFT(&nft.NFT{
			ID:    fmt.Sprintf("nft-%d", i),
			Name:  fmt.Sprintf("NFT %d", i),
			Owner: ownerPub,
		})
		require.NoError(t, err)
	}

	owner := base64.StdEncoding.EncodeToString(ownerPub)

	// limit=2&offset=1 -> exactly two rows. The in-memory repo iterates a Go
	// map (nondeterministic order); SQLite carries the stable rowid order and
	// is covered by TestNFTRepository_GetNFTsByOwnerPaged. Here we assert the
	// count and that both rows come from the owned set.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/list?owner="+owner+"&limit=2&offset=1", nil)
	rr := httptest.NewRecorder()
	handler.List(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	var page []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &page))
	require.Len(t, page, 2)
	for _, n := range page {
		require.Contains(t, []string{"nft-0", "nft-1", "nft-2", "nft-3", "nft-4"}, n.ID)
	}

	// An oversized ?limit= is clamped to maxNFTListLimit (100), not honored.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nft/list?owner="+owner+"&limit=99999", nil)
	rr = httptest.NewRecorder()
	handler.List(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &page))
	require.Len(t, page, 5, "all 5 owned NFTs returned; oversized limit clamped but not fatal")
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
