package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
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
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

type DBConfig struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
}

type APIConfig struct {
	Key string `mapstructure:"key"`
}

func Load() (*Config, error) {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "./logs")
	viper.SetDefault("db.type", "sqlite")
	viper.SetDefault("db.path", "./data/aurora.db")
	// REST API per-client rate limiting (v1.19). Disabled by default; operators
	// opt in so enabling it can never silently break existing traffic.
	viper.SetDefault("api.rateLimit.enabled", false)
	viper.SetDefault("api.rateLimit.requests", 120)
	viper.SetDefault("api.rateLimit.window", time.Minute)
	// Oracle scheduler poll cadence (v1.21). The scheduler's own per-source
	// interval still gates when each feed is due; this is how often it checks.
	viper.SetDefault("oracle.scheduler.checkInterval", time.Second)

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

// RateLimitEnabled reports whether per-client API rate limiting is enabled
// (v1.19). Defaults to off; enable via api.rateLimit.enabled=true.
func RateLimitEnabled() bool {
	return viper.GetBool("api.rateLimit.enabled")
}

// RateLimitRequests returns the per-client request budget (default 120).
func RateLimitRequests() int {
	return viper.GetInt("api.rateLimit.requests")
}

// RateLimitWindow returns the rate-limit window (default 1 minute).
func RateLimitWindow() time.Duration {
	return viper.GetDuration("api.rateLimit.window")
}

// OracleSchedulerCheckInterval returns how often the oracle fetch scheduler
// polls sources for due feeds (default 1s). Per-source Interval still controls
// when each feed is due.
func OracleSchedulerCheckInterval() time.Duration {
	return viper.GetDuration("oracle.scheduler.checkInterval")
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
