package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnknownRoutesAnswerInTheAPIEnvelope: a mistyped path used to receive
// chi's plain-text "404 page not found", which a client parsing the JSON error
// envelope cannot read. The router answers with the same shape as every
// handler, and the reference lists the code.
func TestUnknownRoutesAnswerInTheAPIEnvelope(t *testing.T) {
	mux := newRouter(&Server{})
	for _, path := range []string{"/does-not-exist", "/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		require.Equalf(t, http.StatusNotFound, rr.Code, "path %s", path)
		assert.Equalf(t, "application/json", rr.Header().Get("Content-Type"), "path %s", path)

		var body struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		require.NoErrorf(t, json.Unmarshal(rr.Body.Bytes(), &body), "path %s body %q", path, rr.Body.String())
		assert.Equalf(t, "NOT_FOUND", body.Code, "path %s", path)
	}
}

// TestKnownPathWithWrongMethodAnswersInTheAPIEnvelope: chi's default 405 has an
// empty body, so a client parsing the envelope saw a failed decode instead of a
// method error.
func TestKnownPathWithWrongMethodAnswersInTheAPIEnvelope(t *testing.T) {
	mux := newRouter(&Server{})
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "METHOD_NOT_ALLOWED", body.Code)
}
