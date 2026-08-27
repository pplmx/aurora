package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain isolates $HOME so Load()'s config-file lookup (added in TASK-094:
// $HOME/aurora.toml then ./config/aurora.toml) never picks up a developer's
// real ~/aurora.toml — the defaults tests must run against a clean baseline.
func TestMain(m *testing.M) {
	if os.Getenv("AURORA_TEST_KEEP_HOME") == "" {
		_ = os.Setenv("HOME", os.TempDir())
	}
	os.Exit(m.Run())
}

// resetViper clears viper state so each test starts from a known baseline.
func resetViper() {
	viper.Reset()
}

// captureStdout captures stdout for the duration of fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	require.NoError(t, w.Close())
	os.Stdout = orig
	return <-done
}

// setDevEnv forces AURORA_ENV=development in a way that does not leak across tests.
func setDevEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AURORA_ENV", "development")
}

func TestLoad_Defaults(t *testing.T) {
	resetViper()
	setDevEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "./logs", cfg.Log.Path)
	assert.Equal(t, "sqlite", cfg.DB.Type)
	assert.Equal(t, "./data/aurora.db", cfg.DB.Path)
	assert.NotEmpty(t, cfg.API.Key, "dev mode should auto-generate an API key")
}

// TestLoad_ReadsConfigFile is the ISS-087/TASK-094 regression: cmd/api's only
// config path (Load) never called ReadInConfig, so config/aurora.toml values
// were silently ignored by the API binary. Load must now honor a TOML file
// found via the documented lookup order. The test drives $HOME -> temp dir
// (exactly the production mechanism: $HOME/aurora.toml) because go test's cwd
// is the package dir, so the repo's ./config/aurora.toml is never in scope.
// Note: api.key is deliberately absent from the fixture — the API key is an
// env-only mechanism (AURORA_API_KEY, v1.73), so a config-file key is NOT a
// source and must not be asserted either way.
func TestLoad_ReadsConfigFile(t *testing.T) {
	resetViper()
	setDevEnv(t)

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "aurora.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
[server]
host = "127.0.0.1"
port = 9099

[log]
level = "debug"
path = "/tmp/aurora-logs"

[db]
type = "sqlite"
path = "/tmp/aurora-data/aurora.db"
`), 0o644))
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host, "server.host from config file must be honored")
	assert.Equal(t, 9099, cfg.Server.Port, "server.port from config file must be honored")
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "/tmp/aurora-logs", cfg.Log.Path, "log.path value from config file must be honored")
	assert.Equal(t, "sqlite", cfg.DB.Type)
	assert.Equal(t, "/tmp/aurora-data/aurora.db", cfg.DB.Path,
		"db.path from config file must be honored (previously only the CLI read it)")
}

// TestLoad_MalformedConfigFileFails loudly: a file that exists but cannot be
// parsed must not silently fall back to defaults (that is exactly the silent
// misconfiguration class ISS-087 forbids).
func TestLoad_MalformedConfigFileFails(t *testing.T) {
	resetViper()
	setDevEnv(t)

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "aurora.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("[unclosed\nkey = = value"), 0o644))
	t.Setenv("HOME", tmp)

	_, err := Load()
	require.Error(t, err, "unparseable config file must fail Load, not run on defaults")
	assert.Contains(t, err.Error(), "aurora.toml", "error should name the offending config file")
}

// TestLoad_EnvKeyStillWinsWithConfigFilePresent guards the precedence after the
// config-file read: AURORA_API_KEY (the documented, env-only key mechanism)
// must still resolve even when a $HOME/aurora.toml exists and introduces other
// settings into the same viper instance.
func TestLoad_EnvKeyStillWinsWithConfigFilePresent(t *testing.T) {
	resetViper()
	setDevEnv(t)

	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "aurora.toml"), []byte(`
[server]
port = 9123
`), 0o644))
	t.Setenv("HOME", tmp)
	t.Setenv("AURORA_API_KEY", "live_secure_random_value_xyz_123")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "live_secure_random_value_xyz_123", cfg.API.Key,
		"AURORA_API_KEY env must be honored alongside a config file")
	require.Equal(t, 9123, cfg.Server.Port, "config file and env should coexist")
}

func TestLoad_OverridesViaViperSet(t *testing.T) {
	resetViper()
	setDevEnv(t)

	viper.Set("server.host", "127.0.0.1")
	viper.Set("server.port", 9090)
	viper.Set("log.level", "debug")
	viper.Set("log.path", "/var/log/aurora")
	viper.Set("db.type", "postgres")
	viper.Set("db.path", "/var/lib/aurora/db")
	viper.Set("api.key", "custom-dev-key")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "/var/log/aurora", cfg.Log.Path)
	assert.Equal(t, "postgres", cfg.DB.Type)
	assert.Equal(t, "/var/lib/aurora/db", cfg.DB.Path)
	assert.Equal(t, "custom-dev-key", cfg.API.Key)
}

func TestLoad_GeneratesDevKeyWhenMissing(t *testing.T) {
	resetViper()
	setDevEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Confirm the generated key is a valid base64-url encoded 32-byte value.
	decoded, derr := base64.URLEncoding.DecodeString(cfg.API.Key)
	require.NoError(t, derr)
	assert.Len(t, decoded, 32)
}

func TestLoad_DevEmptyKeyPrintsGeneratedKey(t *testing.T) {
	resetViper()
	setDevEnv(t)
	// viper default for api.key is "" → Load() should auto-generate and print.

	out := captureStdout(t, func() {
		cfg, err := Load()
		require.NoError(t, err)
		assert.NotEmpty(t, cfg.API.Key)
	})
	assert.Contains(t, out, "Generated development API key:")
}

func TestLoad_ProductionRequiresKey(t *testing.T) {
	resetViper()
	t.Setenv("AURORA_ENV", "production")
	// No api.key set.

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t,
		errors.Is(err, ErrMissingAPIKey) || strings.Contains(err.Error(), "AURORA_API_KEY"),
		"expected missing-key error, got: %v", err)
}

func TestLoad_ProductionRejectsInsecureKeys(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{"default", "aurora-api-key-default"},
		{"changeme", "changeme"},
		{"secret", "secret"},
		{"api-key", "api-key"},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetViper()
			t.Setenv("AURORA_ENV", "production")
			viper.Set("api.key", tc.key)

			cfg, err := Load()
			assert.Error(t, err, "key=%q should be rejected", tc.key)
			assert.Nil(t, cfg)
			// Empty key triggers ErrMissingAPIKey (which is checked first in production);
			// all other insecure keys trigger ErrInsecureAPIKey.
			if tc.key == "" {
				assert.True(t, errors.Is(err, ErrMissingAPIKey),
					"key=%q expected ErrMissingAPIKey, got: %v", tc.key, err)
			} else {
				assert.True(t, errors.Is(err, ErrInsecureAPIKey),
					"key=%q expected ErrInsecureAPIKey, got: %v", tc.key, err)
			}
		})
	}
}

// TestLoad_APIKey_BindsAURORA_APIKeyEnv is the ISS-079 regression: Load() is
// the API server's only config path (cmd/api is its sole caller) and never
// read the documented AURORA_API_KEY variable, so production always failed
// with ErrMissingAPIKey even when the operator had set it (the CLI's
// AutomaticEnv in root.go does not help cmd/api). It must now resolve from
// the env var in both modes, and in dev mode must not silently mint a fresh
// random key that invalidates already-served web pages.
func TestLoad_APIKey_BindsAURORA_APIKeyEnv(t *testing.T) {
	t.Run("production", func(t *testing.T) {
		resetViper()
		t.Setenv("AURORA_ENV", "production")
		t.Setenv("AURORA_API_KEY", "live_secure_random_value_xyz_123")
		cfg, err := Load()
		require.NoError(t, err, "production must start when AURORA_API_KEY is set")
		require.Equal(t, "live_secure_random_value_xyz_123", cfg.API.Key)
	})

	t.Run("development", func(t *testing.T) {
		resetViper()
		setDevEnv(t)
		t.Setenv("AURORA_API_KEY", "dev-fixed-key")
		cfg, err := Load()
		require.NoError(t, err)
		require.Equal(t, "dev-fixed-key", cfg.API.Key, "a set env key must be honored, not regenerated per boot")

		resetViper()
		setDevEnv(t)
		out := captureStdout(t, func() {
			cfg, err := Load()
			require.NoError(t, err)
			require.NotEmpty(t, cfg.API.Key)
		})
		require.NotContains(t, out, "Generated development API key:",
			"with AURORA_API_KEY set, no fresh key must be printed")
	})
}

func TestLoad_ProductionAcceptsSecureKey(t *testing.T) {
	resetViper()
	t.Setenv("AURORA_ENV", "production")
	viper.Set("api.key", "live_secure_random_value_xyz_123")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "live_secure_random_value_xyz_123", cfg.API.Key)
}

func TestLoad_ProductionEnvironmentCaseInsensitive(t *testing.T) {
	resetViper()
	t.Setenv("AURORA_ENV", "PRODUCTION")
	// Empty key should still fail in production.
	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetAPIKey(t *testing.T) {
	resetViper()
	setDevEnv(t)
	viper.Set("api.key", "my-key")

	_, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "my-key", GetAPIKey())
}

func TestGenerateAPIKey_ProducesValidBase64(t *testing.T) {
	key, err := GenerateAPIKey()
	require.NoError(t, err)
	require.NotEmpty(t, key)

	decoded, derr := base64.URLEncoding.DecodeString(key)
	require.NoError(t, derr)
	assert.Len(t, decoded, 32, "generated key should decode to 32 bytes")
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	// Generate several keys; collisions are astronomically unlikely.
	seen := make(map[string]struct{}, 100)
	for range 100 {
		k, err := GenerateAPIKey()
		require.NoError(t, err)
		_, dup := seen[k]
		assert.False(t, dup, "duplicate key generated: %s", k)
		seen[k] = struct{}{}
	}
}

func TestOracleSchedulerCheckInterval_DefaultAndOverride(t *testing.T) {
	resetViper()
	Load() // sets defaults (rate limit, scheduler interval)
	require.Equal(t, time.Second, OracleSchedulerCheckInterval(), "default scheduler check interval should be 1s")

	resetViper()
	viper.Set("oracle.scheduler.checkInterval", 5*time.Second)
	require.Equal(t, 5*time.Second, OracleSchedulerCheckInterval())
}

func TestRateLimitTrustedProxies_DefaultEmptyAndOverride(t *testing.T) {
	resetViper()
	Load()
	// Fail-safe default: no proxy is trusted by default, so the rate limiter
	// keys every client on its true socket peer (ISS-073).
	require.Empty(t, RateLimitTrustedProxies(), "default trusted-proxy list should be empty")

	resetViper()
	viper.Set("api.rateLimit.trustedProxies", []string{"203.0.113.10", "10.0.0.0/8"})
	require.Equal(t, []string{"203.0.113.10", "10.0.0.0/8"}, RateLimitTrustedProxies())
}

// TestRateLimitWindow_NumericSeconds is the regression test for the silent
// rate-limit disable (TASK-110): viper.GetDuration(cast.ToDuration) turns a
// bare integer TOML value like `window = 60` into time.Duration(60) = 60
// NANOSECONDS, so the fixed-window limiter reset its budget on every request
// whenever an operator configured the window as a plain number. Numbers and
// numeric strings are now interpreted as SECONDS.
func TestRateLimitWindow_NumericSeconds(t *testing.T) {
	resetViper()
	setDevEnv(t)
	_, err := Load()
	require.NoError(t, err)

	assert.Equal(t, time.Minute, RateLimitWindow(), "default remains 1 minute")

	// Bare integer (the TOML `window = 60` case) → 60 seconds, not 60ns.
	viper.Set("api.rateLimit.window", 60)
	assert.Equal(t, 60*time.Second, RateLimitWindow())

	// Float integer → seconds.
	viper.Set("api.rateLimit.window", float64(30))
	assert.Equal(t, 30*time.Second, RateLimitWindow())

	// Duration string passes through.
	viper.Set("api.rateLimit.window", "500ms")
	assert.Equal(t, 500*time.Millisecond, RateLimitWindow())

	// Numeric string treated as seconds (also previously broken → 0).
	viper.Set("api.rateLimit.window", "120")
	assert.Equal(t, 120*time.Second, RateLimitWindow())
}
