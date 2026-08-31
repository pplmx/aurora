package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	ErrMissingAPIKey  = errors.New("AURORA_API_KEY environment variable is required in production")
	ErrInsecureAPIKey = errors.New("insecure API key detected; please set a secure AURORA_API_KEY")
)

var insecureKeys = map[string]bool{
	"aurora-api-key-default": true,
	"changeme":               true,
	"secret":                 true,
	"api-key":                true,
	"":                       true,
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
	DB     DBConfig     `mapstructure:"db"`
	API    APIConfig    `mapstructure:"api"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	// WebRoot is the directory (relative to the process CWD) that the API
	// server serves the Web UI from. Defaults to "web" (TASK-181, ISS-177):
	// the README documented a configurable web root long before it existed, so
	// operators running cmd/api from a non-repo CWD (a service dir, a
	// container) had no way to point the file server at their checkout.
	WebRoot string `mapstructure:"webRoot"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

// DBConfig configures the SQLite database location. SQLite is the only
// backend: the old db.type knob (v1.9x) was set-but-never-read and has been
// removed so an operator can't expect a non-existent alternate backend.
type DBConfig struct {
	Path string `mapstructure:"path"`
}

type APIConfig struct {
	Key string `mapstructure:"key"`
}

func Load() (*Config, error) {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.webRoot", "web")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "./logs")
	viper.SetDefault("db.path", "./data/aurora.db")
	// REST API per-client rate limiting (v1.19). Disabled by default; operators
	// opt in so enabling it can never silently break existing traffic.
	viper.SetDefault("api.rateLimit.enabled", false)
	viper.SetDefault("api.rateLimit.requests", 120)
	viper.SetDefault("api.rateLimit.window", time.Minute)
	// Rate limiter trusts X-Forwarded-For / X-Real-IP / True-Client-IP ONLY
	// when the direct TCP peer is on this allow-list of proxies/CDNs
	// (v1.69). Defaults to empty: every client keys on its socket peer, so
	// spoofed forwarded headers can never rotate past the budget.
	viper.SetDefault("api.rateLimit.trustedProxies", []string{})
	// Cross-origin allow-list for API/Web UI responses. Defaults to empty:
	// the gateway only needs same-origin access for its own Web UI, and the
	// API key is embedded in served HTML — a wide/wildcard allow-list would
	// let any page the operator visits read the key (v1.64, TASK-077).
	viper.SetDefault("api.cors.allowedOrigins", []string{})
	// Oracle scheduler poll cadence (v1.21). The scheduler's own per-source
	// interval still gates when each feed is due; this is how often it checks.
	viper.SetDefault("oracle.scheduler.checkInterval", time.Second)

	// Read the optional config file with the same lookup order the CLI uses
	// (AGENTS.md: 1. $HOME/aurora.toml  2. ./config/aurora.toml). Previously
	// Load() never called ReadInConfig, so the API binary silently ignored
	// every [server]/[log]/[db]/[api.rateLimit]/[api.cors]/[oracle.scheduler]
	// value operators placed in config/aurora.toml and ran on hardcoded
	// defaults with no error (ISS-087, TASK-094). A missing file is fine (env
	// + defaults); a present-but-unparseable file must fail loudly instead of
	// silently running on defaults. Env vars (AURORA_API_KEY and friends) still
	// take precedence over the file via AutomaticEnv/BindEnv below.
	viper.SetConfigName("aurora")
	viper.SetConfigType("toml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath("./config")
	if err := loadConfigFileIfPresent(); err != nil {
		return nil, err
	}

	// The documented production mechanism for the API key is the
	// AURORA_API_KEY environment variable (ErrMissingAPIKey names it), but
	// Load() is called only by cmd/api, which never invoked AutomaticEnv the
	// way the CLI's root.go does — and default env matching would look for
	// "API.KEY", not "AURORA_API_KEY". Without this binding, cfg.API.Key was
	// always empty: production started with ErrMissingAPIKey even when the
	// operator set AURORA_API_KEY, and dev mode generated a fresh random key
	// on every boot (printing it to stdout and silently invalidating the key
	// already embedded in served web pages). Bind the documented variable now
	// so both paths receive the operator's key (v1.73, ISS-079).
	viper.AutomaticEnv()
	_ = viper.BindEnv("api.key", "AURORA_API_KEY")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	resolvedKey, err := resolveAPIKey(cfg.API.Key)
	if err != nil {
		return nil, err
	}
	cfg.API.Key = resolvedKey
	// Mirror into viper so GetAPIKey() (used by api.NewServer) returns the same
	// key we just resolved — including the auto-generated dev key.
	viper.Set("api.key", resolvedKey)

	return &cfg, nil
}

func GetAPIKey() string {
	return viper.GetString("api.key")
}

// WebRoot returns the directory the API server serves the Web UI from
// (default "web", see ServerConfig.WebRoot; TASK-181, ISS-177). Served via
// router.go's http.FileServer through config.WebRoot() so a custom
// server.webRoot points the file server anywhere the operator mounted the
// checkout — the documented-but-until-now-nonexistent web root option.
func WebRoot() string {
	return viper.GetString("server.webRoot")
}

// RateLimitEnabled reports whether per-client API rate limiting is enabled
// (v1.19). Defaults to off; enable via api.rateLimit.enabled=true.
func RateLimitEnabled() bool {
	return viper.GetBool("api.rateLimit.enabled")
}

// RateLimitRequests returns the per-client request budget (default 120).
func RateLimitRequests() int {
	return viper.GetInt("api.rateLimit.requests")
}

// DurationSeconds resolves a viper duration key, interpreting bare TOML
// numbers and numeric strings as SECONDS (the TASK-110 rule). fallback is
// returned when the key is absent, negative, or zero.
//
// A plain number in TOML (`foo = 60`) is unmarshaled by viper as an int, and
// viper.GetDuration/cast turns int64(60) into time.Duration(60) = 60
// NANOSECONDS — a 60ns HTTP timeout fails every request, a 10ns rate-limit
// window resets the budget on every request (limiter silently disabled), and
// a 250ns scheduler ticker busy-polls. Only a string like "1m" worked through
// the raw viper path. Numbers, numeric strings, "1m"-style strings and
// Duration values all resolve correctly here (TASK-110 fixed only the first
// key this was applied to; http.timeout, http.rateLimit.window and
// oracle.scheduler.checkInterval carry the same bug until routed through
// this helper — TASK-118, ISS-110).
func DurationSeconds(key string, fallback time.Duration) time.Duration {
	switch v := viper.Get(key).(type) {
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			if n > 0 {
				return time.Duration(n) * time.Second
			}
		}
	case int:
		if v > 0 {
			return time.Duration(v) * time.Second
		}
	case int64:
		if v > 0 {
			return time.Duration(v) * time.Second
		}
	case float64:
		if v > 0 {
			return time.Duration(v) * time.Second
		}
	case time.Duration:
		if v > 0 {
			return v
		}
	}
	return fallback
}

// RateLimitWindow returns the rate-limit window (default 1 minute). See
// DurationSeconds for why bare numbers must not reach viper.GetDuration.
func RateLimitWindow() time.Duration {
	return DurationSeconds("api.rateLimit.window", time.Minute)
}

// RateLimitTrustedProxies returns the CIDRs/single IPs whose forwarded
// client headers the rate limiter may believe (default: empty — no proxy is
// trusted, every client keys on its socket peer). See the middleware docs in
// internal/api/middleware/ratelimit.go for the trust model (v1.69).
func RateLimitTrustedProxies() []string {
	return viper.GetStringSlice("api.rateLimit.trustedProxies")
}

// OracleSchedulerCheckInterval returns how often the oracle fetch scheduler
// polls sources for due feeds (default 1s). Per-source Interval still controls
// when each feed is due. Routed through DurationSeconds so a bare TOML number
// (`checkInterval = 250`) means 250 SECONDS, not 250 nanoseconds — the latter
// would make the scheduler's ticker busy-poll (TASK-118, ISS-110).
func OracleSchedulerCheckInterval() time.Duration {
	return DurationSeconds("oracle.scheduler.checkInterval", time.Second)
}

// AllowedCORSOrigins returns the origins allowed to read API/Web UI
// responses cross-origin (default: empty — same-origin only). The Web UI is
// served by the gateway itself, so it needs no CORS header; a wildcard or
// broad allow-list would expose the API key that is embedded in served HTML
// to any page the operator visits (v1.64, TASK-077).
func AllowedCORSOrigins() []string {
	return viper.GetStringSlice("api.cors.allowedOrigins")
}

// loadConfigFileIfPresent reads the optional TOML config file that Load()
// located via SetConfigName/AddConfigPath. A missing file is not an error
// (defaults and env stand alone); a file that exists but cannot be parsed is
// an error so an operator's config mistake surfaces at boot instead of
// silently running on defaults.
func loadConfigFileIfPresent() error {
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("failed to load config file %s: %w", viper.ConfigFileUsed(), err)
	}
	return nil
}

func GenerateAPIKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate API key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(key), nil
}

// resolveAPIKey validates the supplied key (or generates a dev one when missing)
// and returns the effective key to use.
//
// In production, an empty or known-insecure key is rejected with an error.
// In any other environment, an empty key triggers generation of a random 32-byte
// key (printed once to stdout) so local development just works.
func resolveAPIKey(key string) (string, error) {
	isProduction := strings.ToLower(os.Getenv("AURORA_ENV")) == "production"

	if isProduction {
		if key == "" {
			return "", ErrMissingAPIKey
		}
		if insecureKeys[key] {
			return "", ErrInsecureAPIKey
		}
		return key, nil
	}

	if key != "" {
		return key, nil
	}

	generated, err := GenerateAPIKey()
	if err != nil {
		return "", fmt.Errorf("generate development API key: %w", err)
	}
	fmt.Printf("Generated development API key: %s\n", generated)
	return generated, nil
}
