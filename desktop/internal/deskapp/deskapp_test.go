package deskapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/blockchain"
)

// resetState isolates a test from the process-wide viper config and the
// blockchain singleton, which is what makes Start() hermetic.
func resetState(t *testing.T) {
	t.Helper()
	viper.Reset()
	blockchain.ResetForTest()
	dir := t.TempDir()
	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(prev)
		blockchain.ResetForTest()
	})
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		mode    Mode
		opts    Flags
		wantErr string
	}{
		{name: "no flags", mode: ModeRun},
		{name: "help", args: []string{"--help"}, mode: ModeHelp},
		{name: "help ignores other flags", args: []string{"-h", "--data-dir", "/nope"}, mode: ModeHelp},
		{name: "version long", args: []string{"--version"}, mode: ModeVersion},
		{name: "version short", args: []string{"-v"}, mode: ModeVersion},
		{name: "web root", args: []string{"--web-root", "/srv/web"}, mode: ModeRun, opts: Flags{WebRoot: "/srv/web"}},
		{name: "data dir", args: []string{"--data-dir", "/var/lib/aurora"}, mode: ModeRun, opts: Flags{DataDir: "/var/lib/aurora"}},
		{name: "both paths", args: []string{"--web-root", "w", "--data-dir", "d"}, mode: ModeRun, opts: Flags{WebRoot: "w", DataDir: "d"}},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: "flag provided but not defined"},
		{name: "positional rejected", args: []string{"extra"}, wantErr: "takes no positional"},
		{name: "positional rejected after version", args: []string{"--version", "extra"}, wantErr: "takes no positional"},
		{name: "missing value", args: []string{"--data-dir"}, wantErr: "flag needs an argument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, opts, err := ParseFlags(tt.args)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.mode, mode)
			require.Equal(t, tt.opts, opts)
		})
	}
}

func TestUsageAndVersionText(t *testing.T) {
	var b strings.Builder
	Usage(&b, "v9.9.9", "2026-10-21T00:00:00Z")
	out := b.String()
	for _, want := range []string{"aurora-desktop [flags]", "--web-root", "--data-dir", "v9.9.9", "2026-10-21T00:00:00Z"} {
		require.Contains(t, out, want)
	}

	version := VersionText("v1.2.3", "built-then")
	require.Contains(t, version, "v1.2.3")
	require.Contains(t, version, "built-then")
	require.Contains(t, version, "Go: go")
}

func TestResolveWebRoot(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "web")
	require.NoError(t, os.MkdirAll(good, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(good, "index.html"), []byte("<!doctype html>"), 0o644))

	empty := filepath.Join(root, "empty")
	require.NoError(t, os.MkdirAll(empty, 0o755))

	got, err := ResolveWebRoot(good, "")
	require.NoError(t, err)
	require.Equal(t, mustAbs(t, good), got)

	got, err = ResolveWebRoot("", good)
	require.NoError(t, err, "the configured web root is the last fallback")
	require.Equal(t, mustAbs(t, good), got)

	_, err = ResolveWebRoot(empty, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not contain index.html")

	_, err = ResolveWebRoot("", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not locate the web UI")
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return abs
}

func TestPrepareDataDir(t *testing.T) {
	dir := t.TempDir()
	path, err := prepareDataDir(dir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "aurora.db"), path)
	require.DirExists(t, dir)

	original := userDataDir
	t.Cleanup(func() { userDataDir = original })
	userDataDir = func() (string, error) { return filepath.Join(dir, "userdata"), nil }

	path, err = prepareDataDir("")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "userdata", "aurora.db"), path)
	require.DirExists(t, filepath.Join(dir, "userdata"))
}

func TestBackupStamp(t *testing.T) {
	got := BackupStamp(time.Date(2026, 10, 21, 13, 4, 5, 0, time.UTC))
	require.Equal(t, "20261021-130405", got)
}

// TestRuntimeServesTheShippedUIOverTheAppScheme is the integration test: it
// boots the real API server against a temp database and drives the actual
// router — the same handler the window's aurora:// scheme calls — proving the
// desktop shell serves the shipped UI with the injected API key and that an
// authenticated REST call works over that handler.
func TestRuntimeServesTheShippedUIOverTheAppScheme(t *testing.T) {
	resetState(t)
	viper.Set("api.key", "desktop-test-key")
	viper.Set("log.path", filepath.Join(t.TempDir(), "logs"))

	webRoot := shippedWebRoot(t)
	require.FileExists(t, filepath.Join(webRoot, "index.html"))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rt, err := Start(ctx, Options{WebRoot: webRoot, DataDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close()) })

	require.Equal(t, webRoot, rt.WebRoot())
	require.FileExists(t, rt.DBPath())

	index := httptest.NewRecorder()
	rt.Handler().ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, index.Code)
	body := index.Body.String()
	require.Contains(t, body, `window.AURORA_API_KEY = "desktop-test-key"`)
	require.Contains(t, body, "<title>Aurora - Dashboard</title>")
	// The aurora:// transport deliberately carries no CSP: WebKitGTK will not
	// run the page's own scripts under any policy on a custom scheme, so the
	// header the gateway sets is stripped (see Runtime.Handler).
	require.Empty(t, index.Header().Get("Content-Security-Policy"))

	// The API rejects an unauthenticated call...
	unauth := httptest.NewRecorder()
	rt.Handler().ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil))
	require.Equal(t, http.StatusUnauthorized, unauth.Code)

	// ...and accepts the key the page receives.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	req.Header.Set("X-API-Key", "desktop-test-key")
	auth := httptest.NewRecorder()
	rt.Handler().ServeHTTP(auth, req)
	require.Equal(t, http.StatusOK, auth.Code, auth.Body.String())

	var rows []json.RawMessage
	require.NoError(t, json.Unmarshal(auth.Body.Bytes(), &rows))
}

// shippedWebRoot resolves ../web from the repository checkout, independent of
// the working directory resetState() switches to.
func shippedWebRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "web"))
}
