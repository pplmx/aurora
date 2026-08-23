package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// realWebDir resolves the shipped web/ directory from this source file's
// location (independent of the process cwd, which tests may redirect).
func realWebDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "web"))
}

// requireServedAsset GETs path from the given handler (via httptest.NewRecorder
// — the sandbox blocks live loopback listeners, see issue-http-loopback) and
// asserts HTTP 200 plus expected substrings. It keeps the v1.8 "served Web UI
// renders with the injected API key" promise honest against the REAL checkout
// assets rather than a synthetic temp-dir fixture.
func requireServedAsset(t *testing.T, h http.Handler, path string, contains ...string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "GET %s status", path)
	body := rr.Body.String()
	for _, want := range contains {
		require.True(t, strings.Contains(body, want),
			"GET %s body should contain %q", path, want)
	}
	return body
}

// TestWebUIServe_RealAssetsInjectedWithAPIKey boots a real HTTP server over
// the actual web/ directory through the same injectAPIKey gateway the REST
// server uses, and confirms:
//   - the index / voting HTML pages carry window.AURORA_API_KEY (so the
//     browser's /api/v1 calls authenticate instead of returning 401);
//   - the shipped JS and CSS assets that the pages reference actually render.
//
// This is the v1.8 "load the served HTML/JS to confirm the Web UI assets
// render with the injected API key" item.
func TestWebUIServe_RealAssetsInjectedWithAPIKey(t *testing.T) {
	webDir := realWebDir()
	info, err := os.Stat(filepath.Join(webDir, "index.html"))
	require.NoError(t, err, "shipped web/index.html must exist")
	require.True(t, info.Mode().IsRegular())

	handler := injectAPIKey(http.FileServer(http.Dir(webDir)), "test-serve-key")

	index := requireServedAsset(t, handler, "/", `window.AURORA_API_KEY = "test-serve-key";`)
	require.True(t, strings.HasPrefix(index, "<!DOCTYPE html>"), "index should be served as HTML")

	voting := requireServedAsset(t, handler, "/voting.html",
		`window.AURORA_API_KEY = "test-serve-key";`)
	require.True(t, strings.HasPrefix(voting, "<!DOCTYPE html>"))

	requireServedAsset(t, handler, "/lottery.html",
		`window.AURORA_API_KEY = "test-serve-key";`)
	requireServedAsset(t, handler, "/token.html",
		`window.AURORA_API_KEY = "test-serve-key";`)
	requireServedAsset(t, handler, "/oracle.html",
		`window.AURORA_API_KEY = "test-serve-key";`)
	requireServedAsset(t, handler, "/blockchain.html",
		`window.AURORA_API_KEY = "test-serve-key";`)
	requireServedAsset(t, handler, "/nft.html",
		`window.AURORA_API_KEY = "test-serve-key";`)

	// The pages reference these assets; they must resolve (not 404).
	requireServedAsset(t, handler, "/js/app.js", "function votingApp()", "dashboardApp", "function tokenApp()", "function oracleApp()", "function blockchainApp()", "function nftApp()")
	requireServedAsset(t, handler, "/css/style.css", "--accent-voting")
}
