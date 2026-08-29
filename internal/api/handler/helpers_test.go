package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/domain/lottery"
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
			name:       "candidate not in session",
			err:        voting.ErrCandidateNotInSession,
			wantStatus: http.StatusBadRequest,
			wantCode:   "CANDIDATE_NOT_IN_SESSION",
		},
		{
			name:       "wrapped domain error",
			err:        errors.Join(token.ErrInsufficientBalance, errors.New("context")),
			wantStatus: http.StatusBadRequest,
			wantCode:   "INSUFFICIENT_BALANCE",
		},
		{
			name:       "seed too short",
			err:        lottery.ErrSeedTooShort,
			wantStatus: http.StatusBadRequest,
			wantCode:   "SEED_TOO_SHORT",
		},
		{
			name:       "no participants",
			err:        lottery.ErrNoParticipants,
			wantStatus: http.StatusBadRequest,
			wantCode:   "NO_PARTICIPANTS",
		},
		{
			name:       "candidate name required",
			err:        voting.ErrCandidateNameRequired,
			wantStatus: http.StatusBadRequest,
			wantCode:   "CANDIDATE_NAME_REQUIRED",
		},
		{
			name:       "session title required",
			err:        voting.ErrSessionTitleRequired,
			wantStatus: http.StatusBadRequest,
			wantCode:   "SESSION_TITLE_REQUIRED",
		},
		{
			name:       "invalid session time",
			err:        voting.ErrInvalidSessionTime,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_SESSION_TIME",
		},
		{
			name:       "amount too large",
			err:        token.ErrAmountTooLarge,
			wantStatus: http.StatusBadRequest,
			wantCode:   "AMOUNT_TOO_LARGE",
		},
		{
			// TASK-121, ISS-113: a valid-base64 wrong-length vote private key is
			// a client error (400), not a server fault (500).
			name:       "invalid vote private key length",
			err:        voting.ErrInvalidPrivateKey,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_PRIVATE_KEY",
		},
		{
			// TASK-122, ISS-114: a session roster naming a candidate twice is a
			// client error (400), never a silently-doubled tally.
			name:       "duplicate candidate in roster",
			err:        voting.ErrDuplicateCandidate,
			wantStatus: http.StatusBadRequest,
			wantCode:   "DUPLICATE_CANDIDATE",
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
			// Classified (non-500) errors echo their message so the client can
			// act on them; unclassified 500 errors must NOT leak the raw error
			// (information disclosure) and return a generic message instead.
			if tt.wantStatus == http.StatusInternalServerError {
				assert.Equal(t, "internal server error", resp.Error)
			} else {
				assert.Equal(t, tt.err.Error(), resp.Error)
			}
		})
	}
}

// TestWriteUseCaseError_DoesNotLeakUnknownError guards the information-
// disclosure contract: an unclassified (500) error must not echo its raw text
// (which can contain SQL fragments, panic messages, or unexpected failure
// details) back to the API client.
func TestWriteUseCaseError_DoesNotLeakUnknownError(t *testing.T) {
	rr := httptest.NewRecorder()
	secret := "connection string leaked: root:password@tcp(10.0.0.1)"
	writeUseCaseError(rr, errors.New(secret))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotContains(t, resp.Error, secret,
		"unclassified error must not leak its raw message to the client")
	assert.Equal(t, "internal server error", resp.Error)
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

// TestDecodeJSON_RejectsTrailingGarbage pins the strict single-value body
// contract (ISS-171): decodeJSON must reject any data after the first JSON
// value. json.Decoder.Decode reads one value and silently ignores the rest, so
// `{"a":1}{"b":2}` or `{...}non-json` used to pass as well-formed — a lax parse
// path that diverged from a strict parse anywhere else.
func TestDecodeJSON_RejectsTrailingGarbage(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"clean object", `{"a":1}`, true},
		{"clean with trailing ws", `{"a":1}   `, true},
		{"two JSON values", `{"a":1}{"b":2}`, false},
		{"value then garbage", `{"a":1}non-json`, false},
		{"value then array", `{"a":1}[1,2]`, false},
		{"invalid json", `not json at all`, false},
		{"empty body", ``, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			var got struct {
				A int `json:"a"`
			}
			ok := decodeJSON(rr, req, &got)
			assert.Equal(t, tc.want, ok, "decodeJSON verdict for %q", tc.body)
			if !tc.want {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.Contains(t, rr.Body.String(), "INVALID_REQUEST")
			}
		})
	}
}
