package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/stretchr/testify/assert"
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

func TestLotteryHandler_History(t *testing.T) {
	handler := &LotteryHandler{repo: &mockLotteryRepo{}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	rr := httptest.NewRecorder()

	handler.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

type mockLotteryRepo struct{}

func (m *mockLotteryRepo) Save(*lottery.LotteryRecord) error              { return nil }
func (m *mockLotteryRepo) GetByID(string) (*lottery.LotteryRecord, error) { return nil, assert.AnError }
func (m *mockLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	return []*lottery.LotteryRecord{}, nil
}
func (m *mockLotteryRepo) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
	return []*lottery.LotteryRecord{}, nil
}
