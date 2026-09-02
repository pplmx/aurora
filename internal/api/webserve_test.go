package api

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// TestWebUIFormLabelsAssociated guards the round-140 accessibility contract:
// every shipped <label> must be programmatically associated with its form
// control — either via a for attribute that resolves to an element id in the
// same page, or by wrapping the control (WCAG 2.1 1.3.1 / 4.1.2). Before this
// guard all 80+ labels across the five form pages were bare siblings of their
// inputs: screen readers announced every field as "edit text, blank", and
// clicking a label did not focus its field. A dangling for= (typo'd id) is
// treated as broken as a missing one.
func TestWebUIFormLabelsAssociated(t *testing.T) {
	webDir := realWebDir()
	pages := []string{"lottery.html", "voting.html", "token.html", "oracle.html", "nft.html"}
	labelRE := regexp.MustCompile(`(?s)<label\b[^>]*>(.*?)</label>`)
	forRE := regexp.MustCompile(`for="([^"]+)"`)
	idRE := regexp.MustCompile(`\bid="([^"]+)"`)

	var problems []string
	for _, page := range pages {
		body, err := os.ReadFile(filepath.Join(webDir, page))
		require.NoError(t, err)
		ids := map[string]bool{}
		for _, m := range idRE.FindAllStringSubmatch(string(body), -1) {
			ids[m[1]] = true
		}
		for _, m := range labelRE.FindAllStringSubmatch(string(body), -1) {
			label, inner := m[0], m[1]
			openTag := label[:strings.IndexByte(label, '>')]
			hasFor := strings.Contains(openTag, `for="`)
			wraps := strings.Contains(inner, "<input") || strings.Contains(inner, "<select") || strings.Contains(inner, "<textarea")
			if !hasFor && !wraps {
				problems = append(problems, page+": label neither has for= nor wraps a control: "+label)
				continue
			}
			for _, fm := range forRE.FindAllStringSubmatch(openTag, -1) {
				if !ids[fm[1]] {
					problems = append(problems, fmt.Sprintf("%s: label for=%q does not resolve to an element id in the page", page, fm[1]))
				}
			}
		}
	}
	require.Empty(t, problems, "shipped form labels must be accessible (TASK-251):\n"+strings.Join(problems, "\n"))
}

// TestWebUIFocusVisible guards the round-140 focus-visibility contract
// (WCAG 2.4.7): the shipped stylesheet must give keyboard focus on form
// controls a visible indicator. The pre-fix CSS cleared the browser's default
// outline on input:focus and replaced it with nothing but a subtile
// border-color change, so a keyboard operator tabbing through a form saw no
// focus marker at all. A relaxed input:focus border rule for mouse users is
// fine as long as :focus-visible draws a strong ring on keyboard focus.
func TestWebUIFocusVisible(t *testing.T) {
	css := requireServedAsset(t, injectAPIKey(http.FileServer(http.Dir(realWebDir())), "test-serve-key"), "/css/style.css")
	require.Contains(t, css, ":focus-visible", "style.css must style :focus-visible (keyboard focus indicator, TASK-252)")
	require.True(t, strings.Contains(css, "outline: 2px solid"), "style.css must draw a visible focus ring (TASK-252)")
	require.False(t, strings.Contains(css, "outline: none"), "style.css must never suppress the focus outline (WCAG 2.4.7; TASK-252)")
}

// TestWebUIResultLiveRegion guards the round-140 accessible-feedback
// contract: every async result/error container (class="result") must carry
// aria-live="polite" so a screen reader announces form submit outcomes
// ("Create Lottery", "Mint", "Cast Vote" ...) instead of silently updating
// the underlying text (WCAG 4.1.3 / APG live-region pattern).
func TestWebUIResultLiveRegion(t *testing.T) {
	webDir := realWebDir()
	pages := []string{"index.html", "lottery.html", "voting.html", "token.html", "oracle.html", "nft.html", "blockchain.html"}
	tagRE := regexp.MustCompile(`<[^>]*>`)
	for _, page := range pages {
		body, err := os.ReadFile(filepath.Join(webDir, page))
		require.NoError(t, err)
		for _, tag := range tagRE.FindAllString(string(body), -1) {
			if strings.Contains(tag, `class="result"`) {
				require.Contains(t, tag, "aria-live",
					"%s: result container %q must announce updates via aria-live (TASK-253, ISS-249)", page, tag)
			}
		}
	}
}

// TestWebUITableHeaderScope guards the round-143 data-table accessibility
// contract: every shipped <th> must declare scope="col" so a screen reader
// announces the column name for each cell value (WCAG 1.3.1 / 4.1.2), and any
// form <p class="form-hint"> must be reachable via aria-describedby so the
// hint is announced with its input.
func TestWebUITableHeaderScope(t *testing.T) {
	webDir := realWebDir()
	pages := []string{"lottery.html", "voting.html", "token.html", "oracle.html", "nft.html", "index.html"}
	thRE := regexp.MustCompile(`<th(?:\s[^>]*)?>`)
	for _, page := range pages {
		body, err := os.ReadFile(filepath.Join(webDir, page))
		require.NoError(t, err)
		for _, th := range thRE.FindAllString(string(body), -1) {
			require.Contains(t, th, `scope="col"`,
				"%s: table header %q must declare scope=col (TASK-256, ISS-252)", page, th)
		}
		// the lottery count hint is the sole prompt; it must pair id+aria-describedby
		if strings.Contains(string(body), `class="form-hint"`) {
			require.Contains(t, string(body), `aria-describedby=`,
				"%s: a form hint exists but no input announces it via aria-describedby", page)
		}
	}
}

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

// TestWebUIJS_BusyGuardPreventsDoubleSubmit executes the SHIPPED web/js/app.js
// in Node with stubbed window/document/fetch and asserts the withBusy guard
// (TASK-177, ISS-175): every guarded app exposes a per-action busy.<name>
// flag, the flag is true while a request is in flight, a re-entrant call is
// swallowed (no second fetch), and the flag clears when the request settles.
// The syntax gate only parses app.js; this runs it, so a regression in the
// guard (a flag never set, a wrapper dropped) fails go test here instead of
// duplicating records in the browser. Skips cleanly without node, mirroring
// TestWebUIJS_SyntaxValid.
func TestWebUIJS_BusyGuardPreventsDoubleSubmit(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js busy-guard check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	// Stub the browser globals app.js needs at load time, count fetch calls,
	// then drive a guarded read action (verifyDraw does a single fetch and
	// no chained refresh): it must flip busy.<name> on, swallow a
	// double-submit (fetch count stays 1), and flip back off on completion.
	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
global.document = { createElement: () => makeEl(), body: { appendChild: () => {} } };
let calls = 0;
let release = null;
global.fetch = (url, init) => {
  calls += 1;
  return new Promise((res) => { release = () => res({ ok: true, json: () => Promise.resolve({ id: 'x1' }) }); });
};

const fs = require('fs');
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

(async () => {
  const app = global.lotteryApp();
  if (typeof app.busy !== 'object' || app.busy.verifyDraw !== false) {
    console.error('busy.verifyDraw must exist and start false'); process.exit(1);
  }
  app.verifyId = 'L1';
  const p1 = app.verifyDraw();
  if (app.busy.verifyDraw !== true) {
    console.error('busy.verifyDraw must be true while in flight'); process.exit(1);
  }
  // The wrapped body starts on a queued microtask; await one so the first
  // fetch has actually fired before we assert on the re-entrant swallowing.
  await Promise.resolve();
  const p2 = app.verifyDraw();           // double-click while in flight
  if (calls !== 1) {
    console.error('re-entrant submit must not fire a second fetch, got ' + calls); process.exit(1);
  }
  release();                              // let the first request settle
  await p1;
  await p2;                               // swallowed call resolves harmlessly
  if (app.busy.verifyDraw !== false) {
    console.error('busy.verifyDraw must clear after settle'); process.exit(1);
  }
  if (calls !== 1) {
    console.error('exactly one fetch must have fired, got ' + calls); process.exit(1);
  }
  console.log('busy-guard-ok');
})().catch((e) => { console.error(e); process.exit(2); });
`
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "busy_guard_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js busy guard failed:\n%s", out)
	require.Contains(t, string(out), "busy-guard-ok", "web/js/app.js withBusy must guard double-submits")
}

// TestWebUIJS_DashboardKeepPriorOnPollFailure executes the SHIPPED web/js/app.js
// in Node with a stub fetch that always rejects, and asserts the keep-prior
// policy (TASK-151) on the dashboard cards: once a real value has been seen
// (this.loaded is true), a transient poll failure must NOT blank the integrity
// / oracle cards to "?" — only a card that never loaded marks itself '?'.
// This is the regression guard for round-140 TASK-234/ISS-232 (the two cards
// originally blanked unconditionally on every 15s poll hiccup while every
// sibling loader kept its value). Skips cleanly without node.
func TestWebUIJS_DashboardKeepPriorOnPollFailure(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js keep-prior check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
global.document = { createElement: () => makeEl(), body: { appendChild: () => {} } };
// Every API call fails: simulates a transient API blip mid-poll.
global.fetch = () => Promise.reject(new Error('api down'));

const fs = require('fs');
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

(async () => {
  // Cards that HAVE seen a real value keep it across the failure.
  const app = global.dashboardApp();
  app.loaded = true;                       // something already answered
  app.stats.integrity = 'OK';
  app.stats.oracle = '3 OK';
  await app.loadBlockchain();
  await app.loadOracleHealth();
  if (app.stats.integrity !== 'OK') { console.error('integrity must keep OK after a transient failure, got ' + app.stats.integrity); process.exit(1); }
  if (app.stats.oracle !== '3 OK') { console.error('oracle must keep 3 OK after a transient failure, got ' + app.stats.oracle); process.exit(1); }

  // Cards that never loaded still mark themselves '?'.
  const fresh = global.dashboardApp();
  fresh.loaded = false;
  await fresh.loadBlockchain();
  await fresh.loadOracleHealth();
  if (fresh.stats.integrity !== '?') { console.error('never-loaded integrity must be ?, got ' + fresh.stats.integrity); process.exit(1); }
  if (fresh.stats.oracle !== '?') { console.error('never-loaded oracle must be ?, got ' + fresh.stats.oracle); process.exit(1); }
  console.log('keep-prior-ok');
})().catch((e) => { console.error(e); process.exit(2); });
`
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "keep_prior_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js keep-prior failed:\n%s", out)
	require.Contains(t, string(out), "keep-prior-ok", "dashboard integrity/oracle cards must keep prior values across transient poll failures")
}

// TestWebUIJS_VotingRetryRecoversCastVote executes the SHIPPED web/js/app.js in
// Node and pins TASK-243/ISS-241: a transient candidates/sessions load failure
// at init must not permanently disable Cast Vote — the in-page Retry button's
// retryLoad() re-runs the loaders, so once the API recovers the candidates
// roster repopulates without a full page reload. Skips cleanly without node.
func TestWebUIJS_VotingRetryRecoversCastVote(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js voting-retry check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
global.document = { createElement: () => makeEl(), body: { appendChild: () => {} } };
// API down at page load: both initial loaders reject.
let apiUp = false;
const candidates = [{ id: 'c1', name: 'Alice' }];
global.fetch = (url) => {
  if (!apiUp) return Promise.reject(new Error('api down'));
  return Promise.resolve({
    ok: true,
    json: () => Promise.resolve(url.includes('/voting/candidates') ? candidates : []),
  });
};

const fs = require('fs');
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

(async () => {
  const app = global.votingApp();
  await app.init();                       // loaders fail -> candidatesFailed
  if (app.candidatesFailed !== true || app.candidates.length !== 0) {
    console.error('initial failure must set candidatesFailed with empty roster'); process.exit(1);
  }

  // API recovers; in-page Retry must repopulate without a reload.
  apiUp = true;
  await app.retryLoad();
  if (app.candidatesFailed !== false) { console.error('retryLoad must clear candidatesFailed'); process.exit(1); }
  if (app.candidates.length !== 1 || app.candidates[0].id !== 'c1') {
    console.error('retryLoad must restore the candidates roster'); process.exit(1);
  }
  console.log('voting-retry-ok');
})().catch((e) => { console.error(e); process.exit(2); });
`
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "voting_retry_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js voting-retry failed:\n%s", out)
	require.Contains(t, string(out), "voting-retry-ok", "voting page retryLoad() must recover Cast Vote without a reload")
}

// TestWebUIJS_LotteryLoadHistoryKeepPrior executes the SHIPPED web/js/app.js in
// Node and pins TASK-250/ISS-246: a transient refresh failure must NOT blank
// draws that already rendered. loadHistory previously set history=[] in its
// catch, so createLottery's follow-up reload after a successful create would
// wipe the visible list on a 20s timeout / API blip — the opposite of the
// dashboard's keep-prior-rows policy (TASK-151). Skips cleanly without node.
func TestWebUIJS_LotteryLoadHistoryKeepPrior(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js lottery keep-prior check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
global.document = { createElement: () => makeEl(), body: { appendChild: () => {} } };
let mode = 'ok';
global.fetch = (url) => {
  return mode === 'ok'
    ? Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'd1', winners: ['a'] }, { id: 'd2', winners: ['b'] }]) })
    : Promise.reject(new Error('api down'));
};

const fs = require('fs');
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

(async () => {
  const app = global.lotteryApp();
  await app.loadHistory();                      // first load succeeds
  if (app.history.length !== 2) { console.error('expected 2 draws after a successful load, got ' + app.history.length); process.exit(1); }
  mode = 'fail';                                // transient blip on the follow-up reload
  await app.loadHistory();
  if (app.history.length !== 2) { console.error('history must keep 2 rows after a transient failure, got ' + app.history.length); process.exit(1); }
  if (!app.historyFailed) { console.error('historyFailed must still flag the failure'); process.exit(1); }

  // A first-load failure (nothing ever rendered) still distinguishes
  // "couldn't load" from a genuinely empty system via historyFailed.
  const fresh = global.lotteryApp();
  mode = 'fail';
  await fresh.loadHistory();
  if (fresh.history.length !== 0) { console.error('never-loaded history must stay empty, got ' + fresh.history.length); process.exit(1); }
  if (!fresh.historyFailed) { console.error('first-load failure must set historyFailed'); process.exit(1); }
  console.log('lottery-keep-prior-ok');
})().catch((e) => { console.error(e); process.exit(2); });
`
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "lottery_keep_prior_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js lottery keep-prior failed:\n%s", out)
	require.Contains(t, string(out), "lottery-keep-prior-ok", "lottery loadHistory must keep prior rows across a transient refresh failure")
}

// TestWebUIJS_BurnRequiresConfirmation executes the SHIPPED web/js/app.js in
// Node and pins ISS-253: the NFT and Token burn actions are permanently
// destructive (the client gated them behind --confirm; the oracle Delete
// confirms in-page) but the web burn forms were single-click. Each burn must
// call confirm() first and abort (no fetch) when the operator declines —
// otherwise a mis-click or a stray Enter on an auto-filled form destroys an
// asset with no undo. Skips cleanly without node.
func TestWebUIJS_BurnRequiresConfirmation(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js burn-confirm check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
global.document = { createElement: () => makeEl(), body: { appendChild: () => {} } };
let calls = 0;
global.fetch = (url, init) => {
  calls += 1;
  return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
};

const fs = require('fs');
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

(async () => {
  // --- NFT burn: declining the confirm must abort with no fetch ---
  let confirmed = true;            // track what the code asked
  global.confirm = (msg) => { confirmed = !!msg; return false; };
  calls = 0;
  const nft = global.nftApp();
  nft.id = 'NFT-1'; nft.owner = 'OWNER'; nft.privateKey = 'k';
  await nft.burn();
  if (!confirmed) { console.error('nft burn must ask for confirmation'); process.exit(1); }
  if (calls !== 0) { console.error('declined nft burn must not fetch, got ' + calls); process.exit(1); }

  // Accepting the confirm proceeds to the burn endpoint.
  calls = 0;
  global.confirm = () => true;
  await nft.burn();
  if (calls !== 1) { console.error('accepted nft burn must fetch exactly once, got ' + calls); process.exit(1); }

  // --- Token burn: same guard ---
  global.confirm = () => false;
  calls = 0;
  const tok = global.tokenApp();
  tok.tokenId = 'T1'; tok.owner = 'o'; tok.burnAmount = '10'; tok.burnPriv = 'k';
  await tok.burn();
  if (calls !== 0) { console.error('declined token burn must not fetch, got ' + calls); process.exit(1); }

  global.confirm = () => true;
  calls = 0;
  await tok.burn();
  if (calls !== 1) { console.error('accepted token burn must fetch exactly once, got ' + calls); process.exit(1); }

  console.log('burn-confirm-ok');
})().catch((e) => { console.error(e); process.exit(2); });
`
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "burn_confirm_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js burn-confirm guard failed:\n%s", out)
	require.Contains(t, string(out), "burn-confirm-ok", "NFT/Token burn must require an explicit confirm before destroying")
}

// TestWebUIJS_DashboardActivityKeepPriorOnTotalFailure executes the SHIPPED
// web/js/app.js in Node and pins TASK-258/ISS-254: the dashboard Recent
// Activity list must survive a poll cycle in which every endpoint fails. The
// stat cards already kept their last-good values on a transient blip (TASK-151/
// TASK-234), but the list rebuilt itself from per-loader return arrays that
// collapse to [] on failure — so one all-fail 15s poll wiped a populated list
// and replaced it with the false "No recent activity" empty-state. The list
// loaders now signal a failed cycle with null (vs [] = success-with-no-rows)
// and refresh() keeps the prior rows when all of them fail. Skips cleanly
// without node.
func TestWebUIJS_DashboardActivityKeepPriorOnTotalFailure(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js dashboard-activity keep-prior check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
global.document = { createElement: () => makeEl(), body: { appendChild: () => {} } };
let mode = 'ok';
global.fetch = (url) => {
  if (mode === 'fail') return Promise.reject(new Error('down'));
  const routes = {
    '/api/v1/lottery/history': () => [{ id: 'L1', winners: [1] }],
    '/api/v1/voting/candidates': () => [],
    '/api/v1/voting/sessions': () => [{ id: 'S1', title: 'S1', status: 'open', candidates: [] }],
    '/api/v1/blockchain/verify': () => ({ valid: true }),
    '/api/v1/oracle/health': () => [{ successes: 1, failures: 0 }]
  };
  const h = routes[url];
  return Promise.resolve({ ok: true, json: () => Promise.resolve(h ? h() : []) });
};

const fs = require('fs');
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

(async () => {
  // A successful cycle renders 2 rows (lottery L1 + session S1).
  const app = global.dashboardApp();
  await app.refresh();
  if (app.activity.length !== 2) { console.error('expected 2 rows after success, got ' + app.activity.length); process.exit(1); }

  // A full poll failure must NOT collapse the list into "No recent activity".
  mode = 'fail';
  await app.refresh();
  if (app.activity.length !== 2) { console.error('all-endpoint failure must keep the prior rows, got ' + app.activity.length); process.exit(1); }

  // A fresh app whose very first load fails stays empty (nothing to keep) —
  // index.html distinguishes this via loaded=false -> "Couldn't load activity".
  const fresh = global.dashboardApp();
  await fresh.refresh();
  if (fresh.activity.length !== 0) { console.error('never-loaded app must stay empty, got ' + fresh.activity.length); process.exit(1); }

  console.log('dashboard-activity-keep-prior-ok');
})().catch((e) => { console.error(e); process.exit(2); });
`
	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "dashboard_activity_keep_prior_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js dashboard activity keep-prior failed:\n%s", out)
	require.Contains(t, string(out), "dashboard-activity-keep-prior-ok", "dashboard Recent Activity must keep prior rows across a total poll failure")
}

