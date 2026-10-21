package deskapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/infra/sqlite"
)

// TestStart_ServesTheEmbeddedUI is the desktop shell's core promise: with NO
// --web-root and NO checkout-relative assets, the window still gets the real
// Web UI, key-injected and CSP-protected, and the API behind it works.
//
// This is what distinguishes the shell from `aurora-api`: the UI travels
// inside the binary (package aurora's //go:embed), so a double-clicked bundle
// whose working directory is unrelated to the repository renders the dashboard
// instead of a 404.
func TestStart_ServesTheEmbeddedUI(t *testing.T) {
	resetState(t)
	viperKey(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// WebRoot empty = use the embedded copy, even though the test's working
	// directory (resetState chdirs to a temp dir) contains no web/ at all.
	rt, err := Start(ctx, Options{DataDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close()) })

	require.Empty(t, rt.WebRoot(), "no directory is served when the UI is embedded")

	// The dashboard, the other pages, an asset, and the vendored Alpine bundle
	// must all resolve from the embedded filesystem.
	for _, path := range []string{"/", "/lottery.html", "/voting.html", "/token.html", "/oracle.html", "/blockchain.html", "/nft.html", "/css/style.css", "/js/app.js", "/vendor/alpine.min.js"} {
		rr := httptest.NewRecorder()
		rt.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, rr.Code, "GET %s from the embedded UI", path)
		require.NotEmpty(t, rr.Body.String(), "GET %s must serve the embedded file", path)
	}

	// /index.html is canonicalized to / by http.FileServer (the same behavior
	// the HTTP gateway has), so the redirect itself is the correct answer.
	canonical := httptest.NewRecorder()
	rt.Handler().ServeHTTP(canonical, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	require.Equal(t, http.StatusMovedPermanently, canonical.Code, "/index.html must redirect to /")
	require.Equal(t, "./", canonical.Header().Get("Location"))

	// The dashboard must be the real page, with the key bootstrap and CSP the
	// HTTP gateway also applies — the desktop path must not be a weaker one.
	index := httptest.NewRecorder()
	rt.Handler().ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	body := index.Body.String()
	require.Contains(t, body, `window.AURORA_API_KEY = "desktop-test-key"`)
	require.Contains(t, body, "<title>Aurora - Dashboard</title>")
	// No CSP on the custom scheme: WebKit refuses to run the page's own
	// scripts under ANY policy there (measured: 'self' + scheme, the origin,
	// and `*` all left the window inert), so the shell strips the header the
	// gateway sets and relies on the transport instead. Serving the policy
	// unchanged is exactly the regression that rendered a dead dashboard.
	require.Empty(t, index.Header().Get("Content-Security-Policy"),
		"the aurora:// transport must not carry a CSP: it blocks the UI's own scripts")

	// The embedded page's own API calls authenticate with that key.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	req.Header.Set("X-API-Key", "desktop-test-key")
	auth := httptest.NewRecorder()
	rt.Handler().ServeHTTP(auth, req)
	require.Equal(t, http.StatusOK, auth.Code, auth.Body.String())

	var rows []json.RawMessage
	require.NoError(t, json.Unmarshal(auth.Body.Bytes(), &rows))
}

// TestStart_ExplicitWebRootWins locks the development path: --web-root serves
// the pages from disk, so a page can be edited and reloaded without rebuilding
// the binary.
func TestStart_ExplicitWebRootWins(t *testing.T) {
	resetState(t)
	viperKey(t)

	webRoot := shippedWebRoot(t)
	require.FileExists(t, filepath.Join(webRoot, "index.html"))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rt, err := Start(ctx, Options{WebRoot: webRoot, DataDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close()) })

	require.Equal(t, webRoot, rt.WebRoot())
}

// TestCreateDraw_ThroughTheAppScheme_ReachesTheDatabase covers the write path,
// not just the reads the window does on load. The window's fetch calls are
// plain POSTs to /api/v1/... over aurora://, so a create must (a) pass the same
// auth the GETs pass, (b) survive the custom-scheme handler, and (c) land in
// the SQLite file the menu actions and the CLI open. A regression that served
// the UI read-only (for example by mounting the filesystem without the API
// routes, or by pointing the window at a different database) would leave the
// dashboard rendering and the app unable to record anything.
func TestCreateDraw_ThroughTheAppScheme_ReachesTheDatabase(t *testing.T) {
	resetState(t)
	viperKey(t)

	dataDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rt, err := Start(ctx, Options{DataDir: dataDir})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close()) })

	body := strings.NewReader(`{"participants":"Alice,Bob,Carol","seed":"desktop-write-seed","winner_count":1}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lottery/create", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "desktop-test-key")

	create := httptest.NewRecorder()
	rt.Handler().ServeHTTP(create, req)
	require.Equal(t, http.StatusOK, create.Code, create.Body.String())

	var draw struct {
		ID       string   `json:"id"`
		Winners  []string `json:"winners"`
		Verified bool     `json:"verified"`
	}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &draw))
	require.NotEmpty(t, draw.ID)
	require.Len(t, draw.Winners, 1)

	// The write must be durable in the desktop's own database, visible to a
	// second connection - the one a menu export or the CLI would open.
	repo, err := sqlite.NewLotteryRepository(rt.DBPath())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	stored, err := repo.GetByID(draw.ID)
	require.NoError(t, err)
	require.Equal(t, draw.Winners, stored.Winners)
	require.True(t, stored.Verified, "the API verifies before persisting")

	// And it shows up in the list the page polls after a create.
	list := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	listReq.Header.Set("X-API-Key", "desktop-test-key")
	rt.Handler().ServeHTTP(list, listReq)
	require.Equal(t, http.StatusOK, list.Code, list.Body.String())
	require.Contains(t, list.Body.String(), draw.ID)
}

// viperKey sets the API key the gateway demands and keeps logs out of the
// repository checkout.
func viperKey(t *testing.T) {
	t.Helper()
	viper.Set("api.key", "desktop-test-key")
	viper.Set("log.path", filepath.Join(t.TempDir(), "logs"))
}
