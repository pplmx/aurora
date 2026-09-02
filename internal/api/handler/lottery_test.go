package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLotteryHandler_Create_InvalidRequest(t *testing.T) {
	handler := &LotteryHandler{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lottery/create", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", resp.Code)
}

func TestLotteryHandler_Create_MissingFields(t *testing.T) {
	handler := &LotteryHandler{}

	reqBody := CreateLotteryRequest{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lottery/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.NotEqual(t, http.StatusOK, rr.Code)
}

func TestLotteryHandler_Get_NotFound(t *testing.T) {
	handler := &LotteryHandler{repo: &mockLotteryRepo{}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestLotteryHandler_Verify_NotFound(t *testing.T) {
	handler := &LotteryHandler{repo: &mockLotteryRepo{}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/nonexistent/verify", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.Verify(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestLotteryHandler_Verify_ValidDraw(t *testing.T) {
	// Build a real draw through the domain service so its VRF proof, output and
	// public key are internally consistent, then confirm the handler reports it
	// valid (v1.31 audit surface).
	svc := lottery.NewService()
	winners, addrs, output, proof, pubKey, err := svc.DrawWinners([]string{"Alice", "Bob", "Charlie"}, "verify-seed", 1)
	assert.NoError(t, err)
	record := lottery.CreateLotteryRecord("verify-seed", []string{"Alice", "Bob", "Charlie"}, winners, addrs, output, proof, pubKey, 1)

	handler := &LotteryHandler{repo: &stubLotteryRepo{record: record}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/"+record.ID+"/verify", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", record.ID)
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.Verify(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		ID    string `json:"id"`
		Valid bool   `json:"valid"`
	}
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, record.ID, resp.ID)
	assert.True(t, resp.Valid)
}

func TestLotteryHandler_History(t *testing.T) {
	handler := &LotteryHandler{repo: &mockLotteryRepo{}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	rr := httptest.NewRecorder()

	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

// TestLotteryHandler_HistoryPagination pins the REST history paging introduced
// so a key-holding caller cannot force an unbounded scan/response, matching
// the token/NFT/oracle read surfaces (TASK-271, ISS-267). A repo returning N
// records must yield at most the requested (clamped) page size, offset-aware.
func TestLotteryHandler_HistoryPagination(t *testing.T) {
	const n = 60 // > defaultLotteryHistoryLimit so the default cap is exercised
	records := make([]*lottery.LotteryRecord, n)
	for i := range records {
		records[i] = &lottery.LotteryRecord{ID: fmt.Sprintf("L%d", i)}
	}
	handler := &LotteryHandler{repo: &fakeManyLotteryRepo{records: records}}

	t.Run("no-flag default caps at defaultLotteryHistoryLimit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
		rr := httptest.NewRecorder()
		handler.History(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		var got []*lottery.LotteryRecord
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
		require.Len(t, got, defaultLotteryHistoryLimit, "no-flag history must be bounded to the default page size")
	})

	t.Run("limit=3 returns 3 records", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history?limit=3", nil)
		rr := httptest.NewRecorder()
		handler.History(rr, req)
		var got []*lottery.LotteryRecord
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
		require.Len(t, got, 3)
		require.Equal(t, "L0", got[0].ID, "handler slices the repo's ordered list without re-sorting")
	})

	t.Run("offset skips the head of the list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history?limit=3&offset=7", nil)
		rr := httptest.NewRecorder()
		handler.History(rr, req)
		var got []*lottery.LotteryRecord
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
		require.Len(t, got, 3)
		require.Equal(t, "L7", got[0].ID, "offset 7 leaves L7..L9 for a limit of 3")
	})

	t.Run("oversized limit clamps", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history?limit=99999", nil)
		rr := httptest.NewRecorder()
		handler.History(rr, req)
		var got []*lottery.LotteryRecord
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
		require.Len(t, got, n, "limit above maxLotteryHistoryLimit = 100 must clamp, then to the available 60 rows")
	})
}

// fakeManyLotteryRepo returns a fixed in-order list so pagination math can be
// asserted.
type fakeManyLotteryRepo struct {
	records []*lottery.LotteryRecord
}

func (m *fakeManyLotteryRepo) Save(*lottery.LotteryRecord) error { return nil }
func (m *fakeManyLotteryRepo) GetByID(string) (*lottery.LotteryRecord, error) {
	return nil, lottery.ErrNotFound
}
func (m *fakeManyLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	return m.records, nil
}
func (m *fakeManyLotteryRepo) GetByBlockHeight(int64) ([]*lottery.LotteryRecord, error) {
	return nil, nil
}

type mockLotteryRepo struct{}

func (m *mockLotteryRepo) Save(*lottery.LotteryRecord) error { return nil }
func (m *mockLotteryRepo) GetByID(string) (*lottery.LotteryRecord, error) {
	return nil, lottery.ErrNotFound
}
func (m *mockLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	return []*lottery.LotteryRecord{}, nil
}

type stubLotteryRepo struct {
	record *lottery.LotteryRecord
}

func (m *stubLotteryRepo) Save(*lottery.LotteryRecord) error { return nil }
func (m *stubLotteryRepo) GetByID(string) (*lottery.LotteryRecord, error) {
	return m.record, nil
}
func (m *stubLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	if m.record == nil {
		return []*lottery.LotteryRecord{}, nil
	}
	return []*lottery.LotteryRecord{m.record}, nil
}
func (m *stubLotteryRepo) GetByBlockHeight(int64) ([]*lottery.LotteryRecord, error) {
	if m.record == nil {
		return nil, nil
	}
	return []*lottery.LotteryRecord{m.record}, nil
}
func (m *mockLotteryRepo) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
	return []*lottery.LotteryRecord{}, nil
}

func TestNewLotteryHandler(t *testing.T) {
	handler := NewLotteryHandler(&mockLotteryRepo{}, lottery.DefaultWinnerCount)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.repo)
}

func TestLotteryHandler_Routes(t *testing.T) {
	handler := NewLotteryHandler(&mockLotteryRepo{}, lottery.DefaultWinnerCount)
	r := chi.NewRouter()
	handler.Routes(r)

	routes := r.Routes()
	assert.NotEmpty(t, routes)
}

func TestLotteryHandler_CreateRequest(t *testing.T) {
	req := CreateLotteryRequest{
		Participants: "A,B,C",
		Seed:         "seed123",
		WinnerCount:  2,
	}

	assert.Equal(t, "A,B,C", req.Participants)
	assert.Equal(t, "seed123", req.Seed)
	assert.Equal(t, 2, req.WinnerCount)
}

func TestLotteryHandler_Get_WithValidID(t *testing.T) {
	mock := &mockLotteryRepoWithData{}
	handler := &LotteryHandler{repo: mock}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/test-id", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-id")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

type mockLotteryRepoWithData struct{}

func (m *mockLotteryRepoWithData) Save(*lottery.LotteryRecord) error { return nil }
func (m *mockLotteryRepoWithData) GetByID(id string) (*lottery.LotteryRecord, error) {
	return &lottery.LotteryRecord{
		ID:           id,
		Participants: []string{"A", "B", "C"},
	}, nil
}
func (m *mockLotteryRepoWithData) GetAll() ([]*lottery.LotteryRecord, error) {
	return []*lottery.LotteryRecord{}, nil
}
func (m *mockLotteryRepoWithData) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
	return []*lottery.LotteryRecord{}, nil
}

// TestLotteryHandler_History_EmptyArray covers the envelope-consistency fix
// (TASK-114): a repo returning no rows gave a nil slice which JSON-encoded as
// null, unlike every other list endpoint which returns []. The handler must
// normalize to [].
func TestLotteryHandler_History_EmptyArray(t *testing.T) {
	handler := &LotteryHandler{repo: &nilLotteryRepo{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	rr := httptest.NewRecorder()
	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "[]\n", rr.Body.String(), "empty history must encode as [], not null")
}

type nilLotteryRepo struct{}

func (m *nilLotteryRepo) Save(*lottery.LotteryRecord) error { return nil }
func (m *nilLotteryRepo) GetByID(string) (*lottery.LotteryRecord, error) {
	return nil, lottery.ErrNotFound
}
func (m *nilLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	return nil, nil // simulates the SQLite no-rows case
}
func (m *nilLotteryRepo) GetByBlockHeight(int64) ([]*lottery.LotteryRecord, error) {
	return nil, lottery.ErrNotFound
}

// TASK-247: the REST lottery create endpoint previously resolved an omitted
// winner_count to a hardcoded 3, while the CLI's omitted -c fell back to the
// configured lottery.defaultCount. A config defaultCount=4 drew 4 winners via
// CLI but 3 via API. The handler now applies the injected configured default,
// so an omitted winner_count must draw the configured number of winners, not
// a hardcoded 3 (the injected value replaces the domain's DefaultWinnerCount
// in the resolution path; the missing-fields test above still expects a create
// without participants/seed to error).
func TestLotteryHandler_Create_UsesInjectedDefaultWinnerCount(t *testing.T) {
	blockchain.ResetForTest()
	defer blockchain.ResetForTest()

	h := NewLotteryHandler(&mockLotteryRepo{}, 5) // configured defaultCount = 5

	body, _ := json.Marshal(CreateLotteryRequest{
		Participants: "Alice,Bob,Charlie,David,Eve",
		Seed:         "default-count-seed",
		WinnerCount:  0, // omitted: must resolve to the injected 5, not 3
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lottery/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
	var resp lotteryapp.LotteryResponse
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp.Winners, 5)
	assert.Equal(t, 5, len(resp.Winners))
	assert.Len(t, resp.WinnerAddresses, 5)
}
