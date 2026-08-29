package api

import (
	"crypto/sha512"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	// Regression: the lottery create POST must send a comma-separated
	// `participants` string and a `winner_count` field (the pre-v1.52 web code
	// sent an array + `count`, which the API cannot decode -> the browser
	// create silently 400'd).
	require.True(t, strings.Contains(js, "winner_count"), "app.js lottery create must use the winner_count field")

	// TASK-133/ISS-125: live surfaces must not go stale. app.js must expose the
	// polling helper and a refresh() on dashboard + oracle, and the two pages
	// must expose a visible refresh control wired to it.
	require.True(t, strings.Contains(js, "function startPolling("), "app.js must expose startPolling")
	require.True(t, strings.Contains(js, "startPolling(this, 15000)"), "app.js must auto-poll dashboard + oracle surfaces")
	dashboard := requireServedAsset(t, handler, "/", `window.AURORA_API_KEY = "test-serve-key";`)
	require.Contains(t, dashboard, `@click="refresh()"`, "dashboard must expose a refresh button")
	oracle := requireServedAsset(t, handler, "/oracle.html", `window.AURORA_API_KEY = "test-serve-key";`)
	require.Contains(t, oracle, `@click="refresh()"`, "oracle page must expose a refresh button")

	requireServedAsset(t, handler, "/css/style.css", "--accent-voting")
}

// TestWebUIVendorAlpine_LocalNoCDN pins the TASK-132/ISS-124 decision: every
// shipped page must load Alpine from the local /vendor copy, never from the
// unpkg CDN. The CDN tag added a third-party supply-chain/interception
// surface and broke the UI whenever the machine was offline or unpkg was
// unreachable. The vendored build is byte-verified against the published
// alpinejs@3.13.5 SRI so a substituted file cannot be committed silently.
func TestWebUIVendorAlpine_LocalNoCDN(t *testing.T) {
	webDir := realWebDir()
	handler := injectAPIKey(http.FileServer(http.Dir(webDir)), "test-serve-key")

	vendored := filepath.Join(webDir, "vendor", "alpine.min.js")
	st, err := os.Stat(vendored)
	require.NoError(t, err, "shipped web/vendor/alpine.min.js must exist")
	require.True(t, st.Mode().IsRegular())
	require.Greater(t, st.Size(), int64(10000), "vendored Alpine must be the real ~40KiB build, not a stub")

	raw, err := os.ReadFile(vendored)
	require.NoError(t, err)
	sum := sha512.Sum384(raw)
	require.Equal(t,
		"BxpSbjbDhVKwnC1UfcjsNEuMuxg4af5IXOaSi1Iq5rASQ/9a7uslhEXbP9UI/fXo",
		base64.StdEncoding.EncodeToString(sum[:]),
		"vendored Alpine must match the published alpinejs@3.13.5 SRI")

	requireServedAsset(t, handler, "/vendor/alpine.min.js", "Alpine")

	for _, page := range []string{"/", "/lottery.html", "/voting.html", "/token.html",
		"/oracle.html", "/blockchain.html", "/nft.html"} {
		body := requireServedAsset(t, handler, page)
		require.Contains(t, body, `src="/vendor/alpine.min.js"`,
			"%s must load Alpine from the local vendor copy", page)
		require.NotContains(t, body, "unpkg.com",
			"%s must not load Alpine from the unpkg CDN", page)
	}
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

// TestWebUINoUnusedHtmx locks the v1.53 contract: the shipped web UI must not
// load the htmx library (TASK-066, ISS-058). Prior to this guard all seven
// pages pulled htmx from https://unpkg.com/htmx.org@1.9.10 even though nothing
// used it (no hx- attributes, no htmx.* calls anywhere in web/), so every page
// carried dead third-party weight that broke offline use, depended on external
// uptime, and widened the supply-chain/provenance surface. The unresolved
// Alpine.js include is still required (Alpine drives all x-* directives), so
// this guard is deliberately scoped to the removed, unused dependency rather
// than banning every external include.
func TestWebUINoUnusedHtmx(t *testing.T) {
	webDir := realWebDir()
	handler := injectAPIKey(http.FileServer(http.Dir(webDir)), "test-serve-key")

	pages := []string{"/", "/lottery.html", "/voting.html", "/token.html",
		"/oracle.html", "/blockchain.html", "/nft.html"}
	for _, page := range pages {
		body := requireServedAsset(t, handler, page)
		require.Falsef(t, strings.Contains(strings.ToLower(body), "htmx"),
			"page %s still references the removed htmx library (TASK-066 regression)", page)
	}
}

// TestWebUIJS_SyntaxValid runs `node --check` over the shipped web/js/app.js.
// All seven pages were rendered by hand-written Alpine components in a single
// file, and Go-side guards can only assert substrings (function names,
// endpoint literals) — a syntactically broken app.js (a dropped brace, a stray
// comma) would have slipped through `go test` and broken every page at runtime
// (ISS-153). The test skips cleanly when node is absent so Go remains the only
// hard dependency for offline/local developers; CI's ubuntu-latest image ships
// Node, so the gate is enforced on every push/PR.
func TestWebUIJS_SyntaxValid(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js syntax check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")
	// node --check parses (and rejects) the file without executing it, so a
	// syntax regression fails the suite here instead of at browser runtime.
	out, err := exec.Command(node, "--check", appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js is not valid JavaScript:\n%s", out)
	require.NotContains(t, string(out), "SyntaxError", "web/js/app.js must contain no SyntaxError")
}

// TestWebUIJS_ApiFetchKeepsAuthHeader executes the SHIPPED web/js/app.js in
// Node with stubbed window/document/fetch and asserts apiFetch always sends
// X-API-Key — even when a call site supplies its own headers (Content-Type
// for JSON bodies). The round-97 apiFetch refactor routed call sites as
// apiFetch(url, {method, headers: {Content-Type}, body}) while apiFetch
// merged defaults with Object.assign({headers: auroraHeaders()}, options):
// the caller's headers key REPLACED the whole headers object, dropping
// X-API-Key and silently turning every web write into a 401 (ISS-160,
// verified live: token/lottery/nft/voting/oracle POSTs all rejected). Reads
// (no options.headers) and DELETEs kept the key, so the UI looked alive but
// was write-dead. The syntax gate above only parses — it never executes a
// fetch — which is exactly why this regression went uncaught for several
// rounds. This test runs real app.js code against a captured fetch init, so
// a future header-merge regression fails go test here. Skips cleanly without
// node, mirroring TestWebUIJS_SyntaxValid.
func TestWebUIJS_ApiFetchKeepsAuthHeader(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js header-merge check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	// Stub the browser globals app.js needs at load time, capture every fetch
	// init the program would send, then assert the key survives each call shape.
	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
global.document = { createElement: () => makeEl(), body: { appendChild: () => {} } };
let captured = null;
global.fetch = (url, init) => { captured = { url: String(url), init }; return Promise.resolve({ ok: true }); };

const fs = require('fs');
// Indirect eval evaluates in the global scope, so app.js's top-level function
// declarations (apiFetch, auroraHeaders, ...) become reachable globals.
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

(async () => {
  const rows = [];
  const grab = async (label, opts) => {
    await global.apiFetch('/api/v1/x', opts);
    const h = (captured && captured.init && captured.init.headers) || {};
    rows.push([label, h['X-API-Key'] || '', h['Content-Type'] || '']);
  };
  await grab('post-json', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' });
  await grab('post-bare', { method: 'POST', body: '{}' });
  await grab('get', undefined);
  await grab('delete', { method: 'DELETE' });
  console.log(JSON.stringify(rows));
  for (const [, key] of rows) { if (key !== 'testkey') process.exit(1); }
})().catch((e) => { console.error(e); process.exit(2); });
`
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "apiFetch_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js apiFetch dropped X-API-Key on a call site:\n%s", out)
	require.Contains(t, string(out), `["post-json","testkey","application/json"]`,
		"apiFetch must merge the API key with caller-supplied headers")
}
