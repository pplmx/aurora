package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	"github.com/pplmx/aurora/internal/domain/oracle"
)

func TestOracleHandler_Sources_Empty(t *testing.T) {
	repo := oracle.NewInmemRepo()
	handler := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/sources", nil)
	rr := httptest.NewRecorder()

	handler.Sources(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, rr.Body.String())
}

func TestOracleHandler_Sources_SnakeCaseJSONContract(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", Name: "Feed", URL: "http://example.com", Enabled: true})
	handler := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/sources", nil)
	rr := httptest.NewRecorder()
	handler.Sources(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Regression: the web page reads data.sources and s.name etc. (snake_case),
	// but the oracle DTOs previously lacked json tags and emitted PascalCase.
	assert.Contains(t, rr.Body.String(), `"sources"`)
	assert.Contains(t, rr.Body.String(), `"name":"Feed"`)
	assert.NotContains(t, rr.Body.String(), `"Sources"`)
}

func TestOracleHandler_Query_SnakeCaseJSONContract(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", URL: "http://example.com", Enabled: true})
	_ = repo.SaveData(&oracle.OracleData{ID: "d1", SourceID: "s1", Value: "100", Timestamp: 123})
	handler := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/query?source=s1&limit=5", nil)
	rr := httptest.NewRecorder()
	handler.Query(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"data"`)
	assert.Contains(t, rr.Body.String(), `"value":"100"`)
	assert.Contains(t, rr.Body.String(), `"timestamp":123`)
	assert.NotContains(t, rr.Body.String(), `"Value"`)
}

func TestOracleHandler_CreateSource_InvalidJSON(t *testing.T) {
	handler := NewOracleHandler(oracle.NewInmemRepo())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/sources", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()
	handler.CreateSource(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOracleHandler_CreateSource_InvalidURL(t *testing.T) {
	handler := NewOracleHandler(oracle.NewInmemRepo())
	body, _ := json.Marshal(map[string]string{"name": "Feed", "url": "://bad"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/sources", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.CreateSource(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOracleHandler_CreateSource_Success(t *testing.T) {
	repo := oracle.NewInmemRepo()
	handler := NewOracleHandler(repo)
	body, _ := json.Marshal(map[string]interface{}{
		"name": "Price Feed", "url": "http://example.com/price", "interval": 30,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/sources", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.CreateSource(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"name":"Price Feed"`)
	var created struct {
		ID string `json:"id"`
	}
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))
	assert.NotEmpty(t, created.ID)
}

func TestOracleHandler_DeleteSource_Success(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", URL: "http://example.com", Enabled: true})
	handler := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/oracle/sources/s1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "s1")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	handler.DeleteSource(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"deleted"`)
}

func TestOracleHandler_DeleteSource_NotFoundIsIdempotent(t *testing.T) {
	handler := NewOracleHandler(oracle.NewInmemRepo())
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/oracle/sources/missing", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	handler.DeleteSource(rr, req)
	// DELETE is idempotent (plain DELETE FROM ... WHERE id = ? with no
	// rows-affected check), so a missing source still returns 200.
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestOracleHandler_SetSourceEnabled(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", URL: "http://example.com", Enabled: false})
	handler := NewOracleHandler(repo)

	body, _ := json.Marshal(map[string]bool{"enabled": true})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/oracle/sources/s1", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "s1")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	handler.SetSourceEnabled(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"enabled":true`)
}

func TestOracleHandler_SetSourceEnabled_MissingBody(t *testing.T) {
	handler := NewOracleHandler(oracle.NewInmemRepo())
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/oracle/sources/s1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "s1")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	handler.SetSourceEnabled(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOracleHandler_SetSourceEnabled_NotFound(t *testing.T) {
	handler := NewOracleHandler(oracle.NewInmemRepo())
	body, _ := json.Marshal(map[string]bool{"enabled": true})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/oracle/sources/missing", bytes.NewBuffer(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	handler.SetSourceEnabled(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestOracleHandler_Health_NilStatsReturnsEmpty(t *testing.T) {
	handler := NewOracleHandler(oracle.NewInmemRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, "[]", rr.Body.String())
}

type fakeOracleStats struct {
	stats []oracleapp.SourceStat
}

func (f *fakeOracleStats) Stats() []oracleapp.SourceStat { return f.stats }

func TestOracleHandler_Health_WithStats(t *testing.T) {
	handler := NewOracleHandler(oracle.NewInmemRepo())
	handler.SetStats(&fakeOracleStats{stats: []oracleapp.SourceStat{
		{SourceID: "s1", Attempts: 3, Successes: 2, Failures: 1},
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var got []oracleapp.SourceStat
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Len(t, got, 1)
	assert.Equal(t, "s1", got[0].SourceID)
	assert.Equal(t, uint64(3), got[0].Attempts)
	assert.Equal(t, uint64(1), got[0].Failures)
}

func TestOracleHandler_Query_Success(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", URL: "http://example.com", Enabled: true})
	_ = repo.SaveData(&oracle.OracleData{ID: "d1", SourceID: "s1", Value: "100"})
	_ = repo.SaveData(&oracle.OracleData{ID: "d2", SourceID: "s1", Value: "200"})

	handler := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/query?source=s1&limit=5", nil)
	rr := httptest.NewRecorder()

	handler.Query(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Response is a struct wrapping the data; just verify it parses as JSON
	var resp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
}

func TestOracleHandler_Query_InvalidLimit(t *testing.T) {
	repo := oracle.NewInmemRepo()
	handler := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/query?source=s1&limit=abc", nil)
	rr := httptest.NewRecorder()

	handler.Query(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
}

func TestOracleHandler_Query_CapsUnboundedLimit(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", URL: "http://example.com", Enabled: true})
	// Seed far more rows than maxQueryLimit so an uncapped handler would
	// return them all; the handler must clamp to maxQueryLimit.
	for i := 0; i < maxQueryLimit*3; i++ {
		_ = repo.SaveData(&oracle.OracleData{ID: "d" + itoa2(i), SourceID: "s1", Value: "x"})
	}

	handler := NewOracleHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/query?source=s1&limit=999999999", nil)
	rr := httptest.NewRecorder()

	handler.Query(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Data []struct {
			ID string
		} `json:"Data"`
	}
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp.Data, maxQueryLimit)
}

// itoa2 is a tiny non-negative int formatter for the cap test's synthetic ids.
func itoa2(n int) string {
	return strconv.Itoa(n)
}

func TestOracleHandler_Fetch_InvalidJSON(t *testing.T) {
	handler := NewOracleHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/fetch", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Fetch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOracleHandler_Fetch_EmptySource(t *testing.T) {
	repo := oracle.NewInmemRepo()
	handler := NewOracleHandler(repo)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/fetch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Fetch(rr, req)

	// Empty source ID -> ErrSourceNotFound -> 404
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// fakeChain records on-chain heights; satisfies oracleapp.ChainInterface so we
// can exercise OracleHandler.SetChain without touching the real blockchain.
type fakeChain struct {
	calls  int
	height int64
}

func (f *fakeChain) AddLotteryRecord(data string) (int64, error) {
	f.calls++
	return f.height, nil
}

// TestOracleHandler_SetChainWiring proves SetChain is safe to call and that the
// Fetch path still routes correctly with a chain attached. A missing source is
// rejected before any network/chain work, so this is fully hermetic.
func TestOracleHandler_SetChainWiring(t *testing.T) {
	repo := oracle.NewInmemRepo()
	handler := NewOracleHandler(repo)
	chain := &fakeChain{height: 42}
	handler.SetChain(chain)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/fetch", bytes.NewBufferString(`{"source":"missing"}`))
	rr := httptest.NewRecorder()
	handler.Fetch(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("fetch of missing source = %d, want 404", rr.Code)
	}
	if chain.calls != 0 {
		t.Fatalf("chain should not be called for a missing source, got %d calls", chain.calls)
	}
}
