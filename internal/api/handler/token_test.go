package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domaintoken "github.com/pplmx/aurora/internal/domain/token"
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

	// Empty request triggers domain validation errors (invalid base64 owner,
	// invalid amount), which are client-side errors now properly classified as 400.
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Error)
	assert.NotEqual(t, "INTERNAL_ERROR", resp.Code)
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

	assert.Equal(t, http.StatusBadRequest, rr.Code)
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

	assert.Equal(t, http.StatusBadRequest, rr.Code)
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

	assert.Equal(t, http.StatusBadRequest, rr.Code)
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

// recordingHistoryService wraps fakeTokenServiceFull but captures the limit
// passed to GetTransferHistory so the handler's cap is observable.
type recordingHistoryService struct {
	fakeTokenServiceFull
	gotLimit  int
	gotOffset int
}

func (r *recordingHistoryService) GetTransferHistory(_ domaintoken.TokenID, _ domaintoken.PublicKey, limit, offset int) ([]*domaintoken.TransferEvent, error) {
	r.gotLimit = limit
	r.gotOffset = offset
	return nil, nil
}

func TestTokenHandler_History_CapsUnboundedLimit(t *testing.T) {
	svc := &recordingHistoryService{}
	handler := NewTokenHandler(svc)

	// owner must be base64 (GetHistory use case decodes it before the service).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history?token_id=t1&owner=YWxpY2U=&limit=999999999", nil)
	rr := httptest.NewRecorder()

	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, maxHistoryLimit, svc.gotLimit, "unbounded ?limit must be capped")
}

func TestTokenHandler_History_CapsUnboundedOffset(t *testing.T) {
	svc := &recordingHistoryService{}
	handler := NewTokenHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history?token_id=t1&owner=YWxpY2U=&offset=999999999", nil)
	rr := httptest.NewRecorder()

	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, maxHistoryOffset, svc.gotOffset, "unbounded ?offset must be capped")
}

func TestTokenHandler_Approve_Success(t *testing.T) {
	handler := NewTokenHandler(fakeTokenServiceFull{})
	body, _ := json.Marshal(map[string]string{
		"token_id":    "t1",
		"owner":       "YWxpY2U=", // "alice"
		"spender":     "Ym9i",     // "bob"
		"amount":      "100",
		"private_key": "c2VjcmV0", // "secret"
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/approve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Approve(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestTokenHandler_Approve_BadJSON(t *testing.T) {
	handler := NewTokenHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/approve", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Approve(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Allowance_Success(t *testing.T) {
	handler := NewTokenHandler(fakeTokenServiceFull{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/allowance?token_id=t1&owner=YWxpY2U=&spender=Ym9i", nil)
	rr := httptest.NewRecorder()

	handler.Allowance(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestTokenHandler_Allowance_MissingParams(t *testing.T) {
	handler := NewTokenHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/allowance?token_id=t1", nil)
	rr := httptest.NewRecorder()

	handler.Allowance(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
