# API Key Security Research: Aurora v1.2

**Project:** Aurora
**Researched:** 2026-04-30
**Focus:** API key runtime validation (hardcoded default key issue)
**Confidence:** HIGH (Go security fundamentals + established patterns)

## Executive Summary

Aurora's current API key implementation has **critical security flaws**:

1. **Hardcoded default key** (`"aurora-api-key-default"`) exposed in source code
2. **Timing-safe vulnerability** — simple string comparison vulnerable to timing attacks
3. **Information leakage** — error message `"invalid api key"` reveals key structure
4. **No production enforcement** — service starts with insecure defaults

**Required actions:**
1. Use `crypto/subtle.ConstantTimeCompare` for key validation
2. Generate secure random API keys on first run
3. Fail fast in production when using insecure defaults
4. Use generic error messages that don't reveal validation details
5. Implement environment-based configuration with required secrets

---

## Current Implementation Analysis

### Problem: config.go (Line 40)

```go
viper.SetDefault("api.key", "aurora-api-key-default")
```

**Issues:**
- Default key is a predictable string visible in source
- Anyone with source access knows the default key
- Service may run in production without changing the key

### Problem: auth.go (Line 12-13)

```go
if key != apiKey {
    writeUnauthorized(w, "invalid api key")
}
```

**Issues:**
- String comparison (`!=`) is vulnerable to timing attacks
- Error message reveals that a valid key format was received (leaks key structure)
- Attacker can detect if their key format is correct

---

## 1. API Key Generation and Validation Patterns

### Secure Key Generation

API keys should be:
- **Cryptographically random** — use `crypto/rand`, not `math/rand`
- **Minimum 32 bytes** — 256 bits of entropy for HMAC/signing
- **High-entropy output** — encode as base64 or hex

```go
// SECURE: Generate 32-byte API key
func GenerateAPIKey() (string, error) {
    key := make([]byte, 32)
    if _, err := rand.Read(key); err != nil {
        return "", fmt.Errorf("generate API key: %w", err)
    }
    return base64.URLEncoding.EncodeToString(key), nil
}
```

### Constant-Time Comparison

**CRITICAL:** Always use `crypto/subtle.ConstantTimeCompare` to prevent timing attacks.

```go
import "crypto/subtle"

// SECURE: Timing-safe comparison
func validateAPIKey(provided, expected string) bool {
    if len(provided) != len(expected) {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
```

**Why this matters:**
- Regular string comparison (`!=`) returns early on first mismatched byte
- Attacker measures response time to deduce each byte of the key
- Constant-time comparison takes the same time regardless of where mismatch occurs

**Benchmark difference:**
```
String comparison: ~50ns (varies with mismatched position)
ConstantTimeCompare: ~100ns (constant regardless of input)
```

### Improved Auth Middleware

```go
// internal/api/middleware/auth.go
package middleware

import (
    "crypto/subtle"
    "encoding/json"
    "net/http"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            provided := r.Header.Get("X-API-Key")
            
            // Constant-time validation
            if !validateKey(provided, apiKey) {
                // GENERIC ERROR: Don't reveal if key format was correct
                writeUnauthorized(w)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Timing-safe key comparison
func validateKey(provided, expected string) bool {
    if len(provided) != len(expected) {
        return false
    }
    if expected == "" {
        return false // Reject empty keys
    }
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// Generic unauthorized response - no information leakage
func writeUnauthorized(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnauthorized)
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "code":    "UNAUTHORIZED",
            "message": "authentication required",
        },
    })
}
```

---

## 2. Enforcing Key Rotation in Production

### Pattern: Required Secret Validation

**Never start a production service with insecure defaults.**

```go
// internal/config/config.go
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
    ErrMissingAPIKey    = errors.New("AURORA_API_KEY environment variable is required")
    ErrInsecureAPIKey   = errors.New("API key appears to be a default or insecure value")
)

// Known insecure keys that must never be used in production
var insecureKeys = map[string]bool{
    "aurora-api-key-default": true,
    "changeme":               true,
    "secret":                 true,
    "api-key":                true,
    "":                       true,
}

func Load() (*Config, error) {
    // ... existing defaults ...
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    // NOTE: No default for api.key - must be provided
    
    // Environment takes precedence
    viper.AutomaticEnv()
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    
    // Production validation
    if err := validateAPIKeySecurity(cfg.API.Key); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}

// Validate API key security
func validateAPIKeySecurity(key string) error {
    // Check if running in production mode
    isProduction := strings.ToLower(os.Getenv("AURORA_ENV")) == "production"
    
    if key == "" {
        if isProduction {
            return ErrMissingAPIKey
        }
        // Development: generate a temporary key
        generated, err := generateDevKey()
        if err != nil {
            return fmt.Errorf("generate dev key: %w", err)
        }
        // In production, we'd return ErrMissingAPIKey instead
        return nil
    }
    
    // Check for known insecure keys
    if insecureKeys[strings.ToLower(key)] {
        return ErrInsecureAPIKey
    }
    
    // Check key length (minimum entropy)
    if len(key) < 16 {
        return fmt.Errorf("API key too short (minimum 16 characters)")
    }
    
    return nil
}

// Generate a development key (never use in production)
func generateDevKey() (string, error) {
    key := make([]byte, 32)
    if _, err := rand.Read(key); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(key), nil
}
```

### Key Rotation Strategy

```go
// Rotating API keys without downtime
type APIKeyManager struct {
    currentKey  string
    previousKey string // Allow grace period for rotation
    gracePeriod time.Duration
}

// NewAPIKeyManager creates a manager that supports key rotation
func NewAPIKeyManager(currentKey string, gracePeriod time.Duration) *APIKeyManager {
    return &APIKeyManager{
        currentKey:  currentKey,
        previousKey: "", // Set when rotation starts
        gracePeriod: gracePeriod,
    }
}

// RotateKey rotates to a new key, keeping old key valid during grace period
func (m *APIKeyManager) RotateKey(newKey string) {
    m.previousKey = m.currentKey
    m.currentKey = newKey
    
    // In a real implementation, schedule clearing previousKey after gracePeriod
}

// ValidateKey checks against current AND previous key
func (m *APIKeyManager) ValidateKey(provided string) bool {
    return validateKey(provided, m.currentKey) ||
           (m.previousKey != "" && validateKey(provided, m.previousKey))
}
```

---

## 3. Secure Storage of API Keys

### Environment Variables (Recommended for Containers/Kubernetes)

```yaml
# kubernetes/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: aurora-secrets
type: Opaque
stringData:
  aurora-api-key: "your-secure-random-key-here"
```

```yaml
# kubernetes/deployment.yaml
spec:
  containers:
  - name: aurora
    env:
    - name: AURORA_API_KEY
      valueFrom:
        secretKeyRef:
          name: aurora-secrets
          key: aurora-api-key
```

### Configuration File with Permissions (Development Only)

```toml
# config/aurora.toml - NEVER commit this file with real keys!

[api]
# In production, set via environment variable: AURORA_API_KEY
key = "your-secure-key-here"
```

```bash
# Set restrictive permissions
chmod 600 config/aurora.toml
```

### Vault/Secrets Manager Integration (Enterprise)

```go
// internal/config/secrets.go
package config

import (
    "context"
    "errors"
    "fmt"
)

// SecretProvider interface for external secrets
type SecretProvider interface {
    GetSecret(ctx context.Context, key string) (string, error)
}

// HashiCorp Vault provider
type VaultProvider struct {
    address string
    token   string
    path    string
}

func (v *VaultProvider) GetSecret(ctx context.Context, key string) (string, error) {
    // Implementation using hashicorp/vault/api
    // return client.KVv2("secret").Get(ctx, v.path+"/"+key)
    return "", errors.New("not implemented")
}

// AWS Secrets Manager provider
type AWSProvider struct {
    region string
}

func (a *AWSProvider) GetSecret(ctx context.Context, key string) (string, error) {
    // Implementation using aws-sdk-go-v2/service/secretsmanager
    return "", errors.New("not implemented")
}
```

---

## 4. Error Responses That Don't Leak Information

### Security Principle: Fail Closed, Say Nothing

```go
// CURRENT (INSECURE) - reveals key format was correct
writeUnauthorized(w, "invalid api key")

// SECURE - no information about validation details
writeUnauthorized(w) // "authentication required"
```

### Response Comparison

| Scenario | Insecure Response | Secure Response |
|----------|-------------------|-----------------|
| Missing key | `"missing api key"` | `"authentication required"` |
| Wrong key | `"invalid api key"` | `"authentication required"` |
| Key format valid | (attacker learns format) | `"authentication required"` |
| Key length valid | (attacker learns length) | `"authentication required"` |

### Implementation

```go
// Generic authentication error - same response for all failures
func writeUnauthorized(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("WWW-Authenticate", `Basic realm="aurora"`) // For browser clients
    w.WriteHeader(http.StatusUnauthorized)
    
    // Same error for all auth failures - no information leakage
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "code":    "UNAUTHORIZED",
            "message": "authentication required",
        },
    })
}
```

### Logging (Server-Side Only)

```go
// Log details server-side for debugging, but don't expose to client
func (m *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    key := r.Header.Get("X-API-Key")
    
    if key == "" {
        log.Printf("[AUTH] missing API key from %s", r.RemoteAddr)
    } else if len(key) < minKeyLength {
        log.Printf("[AUTH] API key too short from %s", r.RemoteAddr)
    } else if !validateKey(key, m.apiKey) {
        log.Printf("[AUTH] invalid API key from %s (length: %d)", r.RemoteAddr, len(key))
        // Log the provided key length (not the value!) for anomaly detection
    }
    
    writeUnauthorized(w)
}
```

---

## 5. Environment-Based Configuration

### Configuration Precedence

```
1. Environment variable (AURORA_API_KEY) - HIGHEST PRIORITY
2. Config file (config/aurora.toml)
3. Generated default (development only)
4. Hardcoded fallback - NEVER in production
```

### Environment Detection

```go
// Detect environment reliably
func getEnvironment() string {
    env := strings.ToLower(os.Getenv("AURORA_ENV"))
    
    switch env {
    case "production", "prod":
        return "production"
    case "staging", "stage":
        return "staging"
    case "development", "dev":
        return "development"
    default:
        // Default to restricted mode for safety
        return "production"
    }
}

func Load() (*Config, error) {
    env := getEnvironment()
    isProduction := env == "production"
    
    // ... load config ...
    
    if isProduction && cfg.API.Key == "" {
        return nil, fmt.Errorf("AURORA_API_KEY environment variable is required in production")
    }
    
    return &cfg, nil
}
```

### Startup Validation

```go
// cmd/api/main.go
func main() {
    cfg, err := config.Load()
    if err != nil {
        // In production, fail hard - no insecure defaults
        if os.Getenv("AURORA_ENV") == "production" {
            log.Fatalf("FATAL: %v", err)
        }
        // In development, log and continue
        log.Printf("WARNING: %v (using development defaults)", err)
    }
    
    // ... rest of main ...
}
```

---

## Recommended Implementation

### Phase 1: Immediate Security Fix

```go
// internal/config/config.go
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
    ErrMissingAPIKey  = errors.New("AURORA_API_KEY environment variable required in production")
    ErrInsecureAPIKey = errors.New("insecure API key detected")
)

// GenerateAPIKey creates a secure random key
func GenerateAPIKey() (string, error) {
    key := make([]byte, 32)
    if _, err := rand.Read(key); err != nil {
        return "", fmt.Errorf("generate API key: %w", err)
    }
    return base64.URLEncoding.EncodeToString(key), nil
}

func Load() (*Config, error) {
    // No hardcoded default for api.key
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    viper.AutomaticEnv()
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    
    // Validate for production
    isProduction := strings.ToLower(os.Getenv("AURORA_ENV")) == "production"
    
    if isProduction {
        if cfg.API.Key == "" {
            return nil, ErrMissingAPIKey
        }
        // Check for known insecure keys
        if cfg.API.Key == "aurora-api-key-default" {
            return nil, ErrInsecureAPIKey
        }
    } else if cfg.API.Key == "" {
        // Development: generate a temporary key
        key, err := GenerateAPIKey()
        if err != nil {
            return nil, err
        }
        cfg.API.Key = key
        fmt.Printf("Generated development API key: %s\n", key)
    }
    
    return &cfg, nil
}
```

### Phase 2: Timing-Safe Comparison

```go
// internal/api/middleware/auth.go
package middleware

import (
    "crypto/subtle"
    "encoding/json"
    "net/http"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            provided := r.Header.Get("X-API-Key")
            
            if !secureCompare(provided, apiKey) {
                writeUnauthorized(w)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

func secureCompare(provided, expected string) bool {
    if len(provided) != len(expected) {
        return false
    }
    if expected == "" {
        return false
    }
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func writeUnauthorized(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "code":    "UNAUTHORIZED",
            "message": "authentication required",
        },
    })
}
```

---

## Security Checklist

| Requirement | Status | Implementation |
|-------------|--------|----------------|
| No hardcoded API key in source | 🔴 TODO | Remove default, require env var in prod |
| Timing-safe comparison | 🔴 TODO | Use `crypto/subtle.ConstantTimeCompare` |
| Generic error messages | 🔴 TODO | Return same message for all auth failures |
| Production fails on insecure config | 🔴 TODO | Validate on startup, exit if invalid |
| Secure key generation | 🔴 TODO | Use `crypto/rand`, not `math/rand` |
| Key rotation support | 🟡 OPTIONAL | Keep previous key valid during grace period |
| Secret manager integration | 🟡 OPTIONAL | Support Vault, AWS Secrets Manager |

---

## Sources

- [Go crypto/subtle package](https://pkg.go.dev/crypto/subtle) - Constant-time comparison
- [OWASP API Security Top 10](https://owasp.org/API-Security/) - API security best practices
- [NIST SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html) - Authentication guidelines
- [12 Factor App: Config](https://12factor.net/config) - Environment-based configuration
