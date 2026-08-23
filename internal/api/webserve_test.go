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
	// The dashboard surfaces an Oracle feed-health summary (v1.37) alongside
	// lottery/voting/ledger stats so an operator can see feed health at a glance.
	require.True(t, strings.Contains(index, "Oracle Feeds"), "dashboard must expose an Oracle Feeds stat card")

	voting := requireServedAsset(t, handler, "/voting.html",
		`window.AURORA_API_KEY = "test-serve-key";`)
	require.True(t, strings.HasPrefix(voting, "<!DOCTYPE html>"))
	// The voting page must expose the full session lifecycle (create -> start ->
	// vote -> end). Prior to v1.34 these controls were CLI/REST-only, so an
	// operator could create but never start/end a session from the browser.
	require.True(t, strings.Contains(voting, "Start Session"), "voting.html must expose a Start Session control")
	require.True(t, strings.Contains(voting, "End Session"), "voting.html must expose an End Session control")

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

	// Guard against the v1.21 regression where token.html and oracle.html were
	// served without loading /js/app.js — the Alpine components (tokenApp /
	// oracleApp) were then undefined and every page interaction silently
	// failed. Every shipped .html page must reference the shared script.
	for _, page := range []string{
		"/lottery.html", "/voting.html",
		"/token.html", "/oracle.html", "/blockchain.html", "/nft.html",
	} {
		requireServedAsset(t, handler, page, `<script src="/js/app.js"></script>`)
	}
	// /index.html 301-redirects to / (directory index); the dashboard is served
	// at "/", which must also load the shared script.
	requireServedAsset(t, handler, "/", `<script src="/js/app.js"></script>`)

	// The pages reference these assets; they must resolve (not 404).
	js := requireServedAsset(t, handler, "/js/app.js", "function votingApp()", "dashboardApp", "function tokenApp()", "function oracleApp()", "function blockchainApp()", "function nftApp()")
	// votingApp must wire the start/end session endpoints (dynamic {id} paths
	// are not fully captured by the endpoint auto-scanner, so assert directly).
	require.True(t, strings.Contains(js, "'/api/v1/voting/session/'") && strings.Contains(js, "'/start'") && strings.Contains(js, "'/end'"),
		"app.js votingApp must wire /api/v1/voting/session/{id}/start and /end")
	requireServedAsset(t, handler, "/css/style.css", "--accent-voting")
}

// moduleNavLinks are the hrefs every shipped page's header nav must contain.
// New module pages are added incrementally (v1.21–v1.28); without a guard the
// older pages (lottery/voting) can silently keep a stale nav that links only
// to the first few modules. This test locks the full navigation set so a
// future module page is wired everywhere or the test fails loudly.
var moduleNavLinks = []string{
	"/", "/lottery.html", "/voting.html", "/token.html",
	"/oracle.html", "/blockchain.html", "/nft.html",
}

func TestWebUINavigation_AllPagesLinkAllModules(t *testing.T) {
	webDir := realWebDir()
	handler := injectAPIKey(http.FileServer(http.Dir(webDir)), "test-serve-key")

	// /index.html 301-redirects to /; the dashboard is the served page at "/".
	pages := []string{"/", "/lottery.html", "/voting.html", "/token.html",
		"/oracle.html", "/blockchain.html", "/nft.html"}
	for _, page := range pages {
		body := requireServedAsset(t, handler, page)
		for _, href := range moduleNavLinks {
			require.Truef(t, strings.Contains(body, `href="`+href+`"`),
				"page %s nav is missing link to %s (stale navigation regression)", page, href)
		}
	}
}
