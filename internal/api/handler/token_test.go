package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pplmx/aurora/internal/domain/token"
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

func TestTokenHandler_Info_MissingQueryParam(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/info", nil)
	rr := httptest.NewRecorder()
	h.Info(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Info_NotFound(t *testing.T) {
	// fakeTokenServiceFull.GetTokenInfo returns (nil,nil), so the use case
	// maps it to ErrTokenNotFound and the handler responds 404.
	h := NewTokenHandler(fakeTokenServiceFull{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/info?token_id=missing", nil)
	rr := httptest.NewRecorder()
	h.Info(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestTokenHandler_Approve_AuditPublishFailed locks the API half of TASK-117 /
// ISS-109: when a token op COMMITS but its post-commit audit publish fails, the
// response must carry a dedicated code (AUDIT_PUBLISH_FAILED) and a message
// that says the write committed and must not be retried — never a generic
// "internal server error" that a client would blindly retry.
func TestTokenHandler_Approve_AuditPublishFailed(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{
		err: fmt.Errorf("%w: %v", token.ErrAuditPublishFailed, errors.New("event store unavailable")),
	})

	// Valid base64 keys (32-byte public, 64-byte private) so the usecase
	// reaches the service instead of failing decodeKey first.
	body, _ := json.Marshal(map[string]string{
		"token_id":    "TEST",
		"owner":       base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)),
		"spender":     base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32)),
		"amount":      "10",
		"private_key": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 64)),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/approve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Approve(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "AUDIT_PUBLISH_FAILED", resp.Code)
	assert.Contains(t, resp.Error, "committed")
	assert.Contains(t, resp.Error, "do not retry")
}

func TestTokenHandler_TransferFrom_InvalidJSON(t *testing.T) {
	handler := NewTokenHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/transfer_from", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()
	handler.TransferFrom(rr, req)
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
	gotToken  token.TokenID
	gotOwner  token.PublicKey
	gotLimit  int
	gotOffset int
}

func (r *recordingHistoryService) GetTransferHistory(tokenID token.TokenID, owner token.PublicKey, limit, offset int) ([]*token.TransferEvent, error) {
	r.gotToken = tokenID
	r.gotOwner = owner
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

// TestTokenHandler_History_DefaultsToServiceLimit pins TASK-261/ISS-257: the
// REST handler's default limit for the no-?limit= query must equal the service
// layer fallback and the CLI's default (50), so the identical query returns the
// same page size from every surface. It was 20 while both the CLI and the
// service used 50 — same op, same params, different result on a token with >20
// events.
func TestTokenHandler_History_DefaultsToServiceLimit(t *testing.T) {
	svc := &recordingHistoryService{}
	handler := NewTokenHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history?token_id=t1&owner=YWxpY2U=", nil)
	rr := httptest.NewRecorder()

	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// 50 is the shared default across the service layer fallback (limit<=0)
	// and the CLI's --limit resolution; the REST default must match it.
	assert.Equal(t, 50, svc.gotLimit,
		"no-?limit query must use the service/CLI default limit (ISS-257)")
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

// TestTokenHandler_History_MissingTokenIsBadRequest pins the contract that
// History validates its subject like its sibling endpoints: a missing/empty
// token_id must be a 400 INVALID_REQUEST, never a silent 200 [] that a client
// typo could read as "no activity" (ISS-163).
func TestTokenHandler_History_MissingTokenIsBadRequest(t *testing.T) {
	svc := &recordingHistoryService{}
	handler := NewTokenHandler(svc)

	for name, query := range map[string]string{
		"missing both":     "",
		"missing token":    "owner=YWxpY2U=",
		"empty token":      "token_id=&owner=YWxpY2U=",
		"empty both":       "token_id=&owner=",
		"empty token only": "token_id=",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history?"+query, nil)
			rr := httptest.NewRecorder()
			handler.History(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code, "case %q", name)
			assert.Contains(t, rr.Body.String(), "INVALID_REQUEST", "case %q", name)
		})
	}
}

func TestTokenHandler_History_OmittedOwnerRequestsWholeTokenStream(t *testing.T) {
	svc := &recordingHistoryService{}
	handler := NewTokenHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history?token_id=t1", nil)
	rr := httptest.NewRecorder()

	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	assert.Equal(t, token.TokenID("t1"), svc.gotToken)
	assert.Empty(t, svc.gotOwner, "an omitted owner must select the token-wide stream")
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

// TestTokenHandler_List_EmptyArray pins the envelope contract: a store with no
// tokens returns [], not null, like every other list endpoint.
func TestTokenHandler_List_EmptyArray(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/list", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "[]\n", rr.Body.String(), "an empty list must encode as []")
}

// tokenListService returns a fixed collection for the listing tests.
type tokenListService struct {
	fakeTokenServiceFull
	tokens     []*token.Token
	lastLimit  int
	lastOffset int
}

func (f *tokenListService) ListTokens(limit, offset int) ([]*token.Token, error) {
	f.lastLimit, f.lastOffset = limit, offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(f.tokens) {
		return nil, nil
	}
	page := f.tokens[offset:]
	if limit > 0 && len(page) > limit {
		page = page[:limit]
	}
	return page, nil
}

// TestTokenHandler_List_Paged pins the collection listing's bounds. It used to
// return every token in one response; like the NFT and lottery listings, it now
// defaults to a page and caps ?limit so a key-holding caller cannot force an
// unbounded response.
func TestTokenHandler_List_Paged(t *testing.T) {
	tokens := []*token.Token{
		token.NewToken("t1", "One", "ONE", token.NewAmount(1), []byte{1}),
		token.NewToken("t2", "Two", "TWO", token.NewAmount(2), []byte{2}),
		token.NewToken("t3", "Three", "THREE", token.NewAmount(3), []byte{3}),
	}
	service := &tokenListService{tokens: tokens}
	h := NewTokenHandler(service)

	list := func(query string) []struct {
		ID string `json:"id"`
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/token/list"+query, nil)
		rr := httptest.NewRecorder()
		h.List(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		var page []struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &page))
		return page
	}

	require.Len(t, list(""), 3, "the default page holds the three tokens")
	page := list("?limit=1&offset=1")
	require.Len(t, page, 1)
	require.Equal(t, "t2", page[0].ID, "offset pages the newest-first list")
	require.Equal(t, 1, service.lastLimit, "the handler must pass the page size into the use case")
	require.Equal(t, 1, service.lastOffset, "the handler must pass the offset into storage")
	require.Len(t, list("?limit=5000"), 3, "an oversized limit is capped, not rejected")
	require.Equal(t, maxTokenListLimit, service.lastLimit, "the handler must cap the page size before storage")
	require.Empty(t, list("?offset=99"))
}
