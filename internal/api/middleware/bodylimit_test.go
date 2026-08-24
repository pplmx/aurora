package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// countHandler reads the (possibly MaxBytesReader-wrapped) body and reports
// its length, or a 413 if the read trips the cap.
func countHandler(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = bytes.NewBufferString("too large")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(strings.TrimSpace(string(b))))
}

// TestBodyLimit_AllowsUnderCap verifies a body below MaxRequestBody passes
// through the middleware unmodified and readably by the handler.
func TestBodyLimit_AllowsUnderCap(t *testing.T) {
	handler := BodyLimit(http.HandlerFunc(countHandler))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"seed":"s"}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, `{"seed":"s"}`, rr.Body.String())
}

// TestBodyLimit_RejectsOverCapStream verifies the middleware prevents a handler
// from ever reading past MaxRequestBody: an oversized stream trips
// MaxBytesReader and the handler observes *http.MaxBytesError.
func TestBodyLimit_RejectsOverCapStream(t *testing.T) {
	handler := BodyLimit(http.HandlerFunc(countHandler))
	big := make([]byte, MaxRequestBody+1024)
	for i := range big {
		big[i] = 'a'
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(big))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rr.Code, "handler must observe the cap as 413")
}
