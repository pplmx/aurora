package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

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
