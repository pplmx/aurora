package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apimw "github.com/pplmx/aurora/internal/api/middleware"
)

// TestCORS_CrossOriginCannotReadInjectedAPIKey is the composition regression
// test for ISS-069 / TASK-077. The two components are safe in isolation — the
// wildcard-CORS middleware and the key-injecting static file server — but
// together they leaked window.AURORA_API_KEY to any cross-origin page: a
// browser fetch of "/" (a CORS-simple GET) could read the HTML that carries
// the API key, then drive every /api/v1 mutation with it. This builds the same
// chain newRouter wires (CORS outer, then injectAPIKey over the real web/ dir)
// and asserts that a disallowed Origin gets NO Access-Control-Allow-Origin, so
// the browser refuses to expose the response to the attacker's page.
func TestCORS_CrossOriginCannotReadInjectedAPIKey(t *testing.T) {
	webDir := realWebDir()
	const key = "test-serve-key"

	handler := apimw.CORS(nil)( // default: same-origin only
		injectAPIKey(http.FileServer(http.Dir(webDir)), key),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://attacker.example")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// The key is still injected for the same-origin UI ...
	require.Contains(t, rr.Body.String(), `window.AURORA_API_KEY = "test-serve-key";`)
	// ... but the cross-origin page must not be told it may read the response.
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"),
		"cross-origin read of the key-bearing HTML must be denied by omission")
	assert.Equal(t, "Origin", rr.Header().Get("Vary"))
}

// TestCORS_ConfiguredOriginReadsUI confirms the operator opt-in still works:
// an origin listed in api.cors.allowedOrigins gets its ACAO echoed so a
// deliberately cross-origin Web UI keeps functioning.
func TestCORS_ConfiguredOriginReadsUI(t *testing.T) {
	webDir := realWebDir()
	const key = "test-serve-key"
	const uiOrigin = "https://operator.ui"

	handler := apimw.CORS([]string{uiOrigin})(
		injectAPIKey(http.FileServer(http.Dir(webDir)), key),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", uiOrigin)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, uiOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rr.Body.String(), `window.AURORA_API_KEY = "test-serve-key";`)
}
