package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesAndEchoesID(t *testing.T) {
	var contextID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = chimiddleware.GetReqID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	headerID := rr.Header().Get(RequestIDHeader)
	require.NotEmpty(t, headerID)
	assert.Equal(t, headerID, contextID)
	assert.LessOrEqual(t, len(headerID), maxRequestIDLength)
}

func TestRequestID_PreservesSafeClientID(t *testing.T) {
	const want = "trace-0123456789abcdef"
	var contextID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = chimiddleware.GetReqID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, want)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, want, rr.Header().Get(RequestIDHeader))
	assert.Equal(t, want, contextID)
}

func TestRequestID_AcceptsMaximumLength(t *testing.T) {
	want := strings.Repeat("a", maxRequestIDLength)
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, want)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, want, rr.Header().Get(RequestIDHeader))
}

func TestRequestID_ReplacesUnsafeOrOversizedClientID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"space", "trace id"},
		{"control", "trace\nid"},
		{"unicode", "trace-请求-id"},
		{"oversized", strings.Repeat("x", maxRequestIDLength+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var contextID string
			var downstreamHeader string
			handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contextID = chimiddleware.GetReqID(r.Context())
				downstreamHeader = r.Header.Get(RequestIDHeader)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(RequestIDHeader, tt.id)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			headerID := rr.Header().Get(RequestIDHeader)
			require.NotEmpty(t, headerID)
			assert.NotEqual(t, tt.id, headerID)
			assert.Equal(t, headerID, contextID)
			assert.Equal(t, headerID, downstreamHeader,
				"downstream code must not see the unvalidated client value")
			assert.LessOrEqual(t, len(headerID), maxRequestIDLength)
		})
	}
}
