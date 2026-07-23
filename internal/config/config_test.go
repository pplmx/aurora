package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
