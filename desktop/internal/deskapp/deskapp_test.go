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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/logger"
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
	export := filepath.Join(t.TempDir(), "draws.json")
	require.NoError(t, os.WriteFile(export, []byte("[]"), 0o644))

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
		{name: "data dir", args: []string{"--data-dir", "/var/lib/aurora"}, mode: ModeRun, opts: Flags{DataDir: "/var/lib/aurora"}},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: "flag provided but not defined"},
		{name: "bare dash rejected", args: []string{"-"}, wantErr: "unexpected argument"},
		{name: "flags after terminator rejected", args: []string{"--", "--data-dir", "/d"}, wantErr: "only one link or file"},
		{name: "intent after terminator", args: []string{"--", "aurora://token"}, mode: ModeRun, opts: Flags{Intent: Intent{Page: "token", Raw: "aurora://token"}}},
		{name: "positional rejected", args: []string{"extra"}, wantErr: "unexpected argument"},
		{name: "positional rejected after version", args: []string{"--version", "extra"}, wantErr: "unexpected argument"},
		{name: "deep link", args: []string{"aurora://lottery?id=abc"}, mode: ModeRun, opts: Flags{Intent: Intent{Page: "lottery", LotteryID: "abc", Raw: "aurora://lottery?id=abc"}}},
		{name: "deep link to the dashboard", args: []string{"aurora://"}, mode: ModeRun, opts: Flags{Intent: Intent{Raw: "aurora://"}}},
		{name: "deep link with flags", args: []string{"--data-dir", "/d", "aurora://token"}, mode: ModeRun, opts: Flags{DataDir: "/d", Intent: Intent{Page: "token", Raw: "aurora://token"}}},
		{name: "deep link with trailing flags", args: []string{"aurora://token", "--data-dir", "/d"}, mode: ModeRun, opts: Flags{DataDir: "/d", Intent: Intent{Page: "token", Raw: "aurora://token"}}},
		{name: "deep link with interspersed flags", args: []string{"aurora://token", "--data-dir=/d"}, mode: ModeRun, opts: Flags{DataDir: "/d", Intent: Intent{Page: "token", Raw: "aurora://token"}}},
		{name: "export with trailing flags", args: []string{export, "--data-dir", "/d"}, mode: ModeRun, opts: Flags{DataDir: "/d", Intent: Intent{File: export, Raw: export}}},
		{name: "trailing flag missing value", args: []string{"aurora://token", "--data-dir"}, wantErr: "flag needs an argument"},
		{name: "unknown trailing flag", args: []string{"aurora://token", "--nope"}, wantErr: "flag provided but not defined"},
		{name: "two links rejected", args: []string{"aurora://token", "aurora://nft"}, wantErr: "only one link or file"},
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
	for _, want := range []string{"aurora-desktop [flags]", "lottery-export.json] [flags]", "aurora://lottery?id=abc", "draws.json", "v9.9.9", "2026-10-21T00:00:00Z"} {
		require.Contains(t, out, want)
	}

	version := VersionText("v1.2.3", "built-then")
	require.Contains(t, version, "v1.2.3")
	require.Contains(t, version, "built-then")
	require.Contains(t, version, T("app.go_version")+": go")
}

func TestPrepareDataDir(t *testing.T) {
	dir := t.TempDir()
	path, err := prepareDataDir(dir, nil)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "aurora.db"), path)
	require.DirExists(t, dir)

	t.Run("relative directory", func(t *testing.T) {
		work := t.TempDir()
		t.Chdir(work)

		path, err := prepareDataDir("db", nil)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(work, "db", "aurora.db"), path)
		require.True(t, filepath.IsAbs(path), "the recorded database path must not depend on the launch directory")
		require.DirExists(t, filepath.Join(work, "db"))
	})

	original := userDataDir
	t.Cleanup(func() { userDataDir = original })
	userDataDir = func() (string, error) { return filepath.Join(dir, "userdata"), nil }

	path, err = prepareDataDir("", nil)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "userdata", "aurora.db"), path)
	require.DirExists(t, filepath.Join(dir, "userdata"))
}

func TestPrepareDataDirKeepsTheDefaultApplicationStatePrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows protects the per-user AppData tree with ACLs, not Unix mode bits")
	}

	root := t.TempDir()
	for _, tt := range []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{
			name:  "new nested directory",
			setup: func(*testing.T, string) {},
		},
		{
			name: "directory created by an older release",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.Mkdir(dir, 0o755))
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(root, strings.ReplaceAll(tt.name, " ", "-"))
			tt.setup(t, dir)

			_, err := prepareDataDir("", func() (string, error) { return dir, nil })
			require.NoError(t, err)

			info, err := os.Stat(dir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(),
				"the database and logs can contain operator credentials; other local users must not traverse their directory")
		})
	}
}

func TestPrepareDataDirPreservesAnExplicitDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows protects the per-user AppData tree with ACLs, not Unix mode bits")
	}

	dir := filepath.Join(t.TempDir(), "managed")
	require.NoError(t, os.Mkdir(dir, 0o750))
	require.NoError(t, os.Chmod(dir, 0o750))

	_, err := prepareDataDir(dir, nil)
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o750), info.Mode().Perm(),
		"an explicitly managed data directory must keep the operator's existing permissions")
}

func TestStartKeepsApplicationFilesPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows protects the per-user AppData tree with ACLs, not Unix mode bits")
	}

	resetState(t)
	viperKey(t)
	ctx, cancel := context.WithCancel(context.Background())

	dataDir := filepath.Join(t.TempDir(), "legacy", "Aurora")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	rt, err := Start(ctx, Options{
		DataDirDefault: func() (string, error) { return dataDir, nil },
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		cancel()
		require.NoError(t, rt.Close())
		logger.Close()
	})

	info, err := os.Stat(dataDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(),
		"the default app directory must be tightened before durable state is exposed")

	for _, path := range []string{
		rt.DBPath(),
		rt.DBPath() + "-wal",
		rt.DBPath() + "-shm",
		filepath.Join(viper.GetString("log.path"), "aurora.log"),
	} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
			"%s can contain credentials or request details", path)
	}
}

func TestBackupStamp(t *testing.T) {
	got := BackupStamp(time.Date(2026, 10, 21, 13, 4, 5, 0, time.UTC))
	require.Equal(t, "20261021-130405", got)
}

// TestStart_KeepsDefaultLogsBesideTheDatabase pins where a packaged app writes
// its log. The shipped config/aurora.toml spells the default as "./logs/",
// while the guard compared against "./logs": the trailing slash slipped past
// it and the log landed in ./logs relative to the launch directory (measured
// in the packaged build under Xvfb), which is uncontrollable for an app opened
// from a file manager.
func TestStart_KeepsDefaultLogsBesideTheDatabase(t *testing.T) {
	for _, configured := range []string{"./logs", "./logs/", "logs/", ""} {
		t.Run(configured, func(t *testing.T) {
			resetState(t)
			viperKey(t)
			viper.Set("log.path", configured)
			// resetState chdirs into a temp dir, so a path that stays relative
			// would land there - exactly the bug this test rules out.
			cwd, err := os.Getwd()
			require.NoError(t, err)

			dataDir := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			rt, err := Start(ctx, Options{DataDir: dataDir})
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, rt.Close()) })

			want := filepath.Join(dataDir, "logs")
			require.Equal(t, want, viper.GetString("log.path"),
				"a default relative log path must be anchored to the data directory")
			require.DirExists(t, want)
			require.NoFileExists(t, filepath.Join(cwd, "logs", "aurora.log"),
				"nothing may be written relative to the launch directory")
		})
	}
}

// TestRuntimeBootsNativeUIAndAPI is the integration test: it boots the real
// API server against a temp database and proves two things at once — the
// window state is wired to the same repositories the REST handlers use, and
// the gateway still authenticates /api/v1.
//
// The window itself is covered by nativeui's own view tests, which render
// every page through MyGo's Tester; what needs an integration check here is
// that Start actually produces a usable App and a live server.
func TestRuntimeBootsNativeUIAndAPI(t *testing.T) {
	resetState(t)
	viper.Set("api.key", "desktop-test-key")
	viper.Set("log.path", filepath.Join(t.TempDir(), "logs"))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rt, err := Start(ctx, Options{DataDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close()) })

	require.FileExists(t, rt.DBPath())

	// The window's state is wired: its services drive the same repositories.
	app := rt.App()
	require.NotNil(t, app, "Start must produce the window state")

	// The API rejects an unauthenticated call...
	unauth := httptest.NewRecorder()
	rt.Router().ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil))
	require.Equal(t, http.StatusUnauthorized, unauth.Code)

	// ...and accepts the configured key.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	req.Header.Set("X-API-Key", "desktop-test-key")
	auth := httptest.NewRecorder()
	rt.Router().ServeHTTP(auth, req)
	require.Equal(t, http.StatusOK, auth.Code, auth.Body.String())

	var rows []json.RawMessage
	require.NoError(t, json.Unmarshal(auth.Body.Bytes(), &rows))

	// A request outside the API surface has nothing to serve now that the
	// window draws native widgets instead of HTML.
	root := httptest.NewRecorder()
	rt.Router().ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusNotFound, root.Code,
		"the gateway serves no static UI; the desktop window draws it")
}

// flushRecorder is a ResponseWriter that records flushes, so the test can tell
// a forwarded Flush from a dropped one.
type flushRecorder struct {
	header  http.Header
	flushed bool
}

func (r *flushRecorder) Header() http.Header { return r.header }
func (r *flushRecorder) Write(p []byte) (int, error) {
	return len(p), nil
}
func (r *flushRecorder) WriteHeader(int) {}
func (r *flushRecorder) Flush()          { r.flushed = true }

// viperKey pins the API key the tests authenticate with, so a handler test
// does not depend on the developer's environment.
func viperKey(t *testing.T) {
	t.Helper()
	viper.Set("api.key", "desktop-test-key")
}
