package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/assert"
)

func TestWriteInternalError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeInternalError(rr)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "internal server error", resp.Error)
	assert.Equal(t, "INTERNAL_ERROR", resp.Code)
}

func TestWriteBadRequest(t *testing.T) {
	rr := httptest.NewRecorder()

	writeBadRequest(rr, "test message")

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "test message", resp.Error)
	assert.Equal(t, "INVALID_REQUEST", resp.Code)
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeError(rr, "custom error", "CUSTOM_CODE", http.StatusForbidden)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "custom error", resp.Error)
	assert.Equal(t, "CUSTOM_CODE", resp.Code)
}

func TestWriteUseCaseError_DomainError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "token not found",
			err:        token.ErrTokenNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "TOKEN_NOT_FOUND",
		},
		{
			name:       "insufficient balance",
			err:        token.ErrInsufficientBalance,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INSUFFICIENT_BALANCE",
		},
		{
			name:       "nft not found",
			err:        nft.ErrNFTNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "NFT_NOT_FOUND",
		},
		{
			name:       "source not found",
			err:        oracle.ErrSourceNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "SOURCE_NOT_FOUND",
		},
		{
			name:       "session not found",
			err:        voting.ErrSessionNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "SESSION_NOT_FOUND",
		},
		{
			name:       "already voted",
			err:        voting.ErrAlreadyVoted,
			wantStatus: http.StatusConflict,
			wantCode:   "ALREADY_VOTED",
		},
		{
			name:       "wrapped domain error",
			err:        errors.Join(token.ErrInsufficientBalance, errors.New("context")),
			wantStatus: http.StatusBadRequest,
			wantCode:   "INSUFFICIENT_BALANCE",
		},
		{
			name:       "unknown error defaults to 500",
			err:        errors.New("something went wrong"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeUseCaseError(rr, tt.err)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			var resp ErrorResponse
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCode, resp.Code)
			assert.Equal(t, tt.err.Error(), resp.Error)
		})
	}
}

func TestClassifyError_NilError(t *testing.T) {
	status, code := classifyError(nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "OK", code)
}

func TestClassifyError_UnknownError(t *testing.T) {
	status, code := classifyError(errors.New("unknown"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "INTERNAL_ERROR", code)
}
