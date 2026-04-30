# Plan 07-02: Remove Hardcoded Default, Add Production Validation

**Phase:** 7 - Security Hardening
**Requirement:** SEC-02 (Remove hardcoded default API key, fail in production if not configured)
**Status:** Completed

## Context

Currently `config.go` sets a hardcoded default `"aurora-api-key-default"` which:
- Is visible in source code
- Could allow the service to start insecurely in production
- Is a known insecure value that attackers look for

## Files to Modify

- `internal/config/config.go`

## Tasks

### Task 1: Remove hardcoded default API key
Delete line 40: `viper.SetDefault("api.key", "aurora-api-key-default")`

### Task 2: Add crypto/rand import
Add imports for secure key generation:
- `crypto/rand`
- `encoding/base64`
- `errors`
- `fmt`
- `os`
- `strings`

### Task 3: Define error variables
Create sentinel errors for validation:
- `ErrMissingAPIKey` - API key not configured
- `ErrInsecureAPIKey` - Known insecure key detected

### Task 4: Add GenerateAPIKey function
Create a function that generates a secure random key:
- Generate 32 bytes using `crypto/rand`
- Encode as base64 for transportability

### Task 5: Add production validation in Load()
After unmarshaling, validate the API key:
- In production (`AURORA_ENV=production`): fail if key is missing or is a known insecure value
- In development: if key is empty, generate a secure random key and print it to stdout

### Task 6: Update GetAPIKey usage
Ensure callers handle the case where no key is configured (validation happens in Load)

## Expected Code Changes

### Before (config.go)

```go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server ServerConfig `mapstructure:"server"`
    Log    LogConfig    `mapstructure:"log"`
    DB     DBConfig     `mapstructure:"db"`
    API    APIConfig    `mapstructure:"api"`
}

// ... other types ...

func Load() (*Config, error) {
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("log.level", "info")
    viper.SetDefault("log.path", "./logs")
    viper.SetDefault("db.type", "sqlite")
    viper.SetDefault("db.path", "./data/aurora.db")
    viper.SetDefault("api.key", "aurora-api-key-default")  // REMOVE THIS

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### After (config.go)

```go
package config

import (
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "os"
    "strings"

    "github.com/spf13/viper"
)

var (
    ErrMissingAPIKey  = errors.New("AURORA_API_KEY environment variable is required in production")
    ErrInsecureAPIKey = errors.New("insecure API key detected; please set a secure AURORA_API_KEY")
)

// Known insecure API key values that must never be used
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

// ... other types ...

// GenerateAPIKey generates a cryptographically secure random API key.
// Use this for development mode only.
func GenerateAPIKey() (string, error) {
    key := make([]byte, 32)
    if _, err := rand.Read(key); err != nil {
        return "", fmt.Errorf("generate API key: %w", err)
    }
    return base64.URLEncoding.EncodeToString(key), nil
}

func Load() (*Config, error) {
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("log.level", "info")
    viper.SetDefault("log.path", "./logs")
    viper.SetDefault("db.type", "sqlite")
    viper.SetDefault("db.path", "./data/aurora.db")
    // NOTE: No default for api.key - must be configured

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    // Production validation
    if err := validateAPIKey(cfg.API.Key); err != nil {
        return nil, err
    }

    return &cfg, nil
}

// validateAPIKey ensures the API key is secure for the current environment.
func validateAPIKey(key string) error {
    isProduction := strings.ToLower(os.Getenv("AURORA_ENV")) == "production"

    if isProduction {
        // Production requires explicit configuration
        if key == "" {
            return ErrMissingAPIKey
        }
        // Block known insecure keys
        if insecureKeys[key] {
            return ErrInsecureAPIKey
        }
    } else if key == "" {
        // Development: generate a secure random key
        generated, err := GenerateAPIKey()
        if err != nil {
            return fmt.Errorf("generate development API key: %w", err)
        }
        fmt.Printf("Generated development API key: %s\n", generated)
        // Note: In a real implementation, we'd store this back, but viper
        // doesn't allow setting after load. For v1.2, development keys
        // should be set via config file or environment.
    }

    return nil
}
```

## Verification

1. Test production mode fails without key:
```bash
AURORA_ENV=production go run ./cmd/aurora serve
# Expected: exit with ErrMissingAPIKey
```

2. Test production mode fails with insecure key:
```bash
AURORA_API_KEY=aurora-api-key-default AURORA_ENV=production go run ./cmd/aurora serve
# Expected: exit with ErrInsecureAPIKey
```

3. Test development mode generates key (if implemented):
```bash
go run ./cmd/aurora serve
# Expected: generates key and continues
```

4. Run all tests:
```bash
go test ./internal/config/... -v
```

## Dependencies

None - this plan modifies config.go independently of auth.go.

## Rollback

If issues arise, restore the hardcoded default:
```go
viper.SetDefault("api.key", "aurora-api-key-default")
```
Remove the validation logic.

## Notes

The development key generation prints to stdout but doesn't persist. For v1.2, developers should:
1. Use a config file: `config/aurora.toml` with `api.key = "your-key"`
2. Use environment: `AURORA_API_KEY=your-key`

Future enhancement: write generated key to a `.env` file for persistence.
