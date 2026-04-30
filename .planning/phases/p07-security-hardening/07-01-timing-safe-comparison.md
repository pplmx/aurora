# Plan 07-01: Timing-Safe API Key Comparison

**Phase:** 7 - Security Hardening
**Requirement:** SEC-01 (API authentication uses constant-time comparison)
**Status:** Completed

## Context

The current auth middleware uses a simple string comparison (`key != apiKey`) which is vulnerable to timing attacks. An attacker can measure response times to deduce the correct API key byte by byte.

## Files to Modify

- `internal/api/middleware/auth.go`

## Tasks

### Task 1: Import crypto/subtle
Add the import for the constant-time comparison package.

### Task 2: Add secureCompare helper function
Create a function that compares keys in constant time:
- Check length first (early exit for obvious mismatches)
- Use `subtle.ConstantTimeCompare` for the actual comparison
- Reject empty keys

### Task 3: Replace vulnerable comparison
Replace `if key != apiKey` with `if !secureCompare(key, apiKey)`

### Task 4: Update writeUnauthorized call
Remove the message parameter since we'll use generic errors (handled in Plan 07-03)

### Task 5: Add tests for secureCompare
Test various scenarios including:
- Matching keys
- Non-matching keys (different length)
- Non-matching keys (same length)
- Empty provided key
- Empty expected key

## Expected Code Changes

### Before (auth.go)

```go
package middleware

import (
    "encoding/json"
    "net/http"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("X-API-Key")
            if key != apiKey {
                writeUnauthorized(w, "invalid api key")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### After (auth.go)

```go
package middleware

import (
    "crypto/subtle"
    "encoding/json"
    "net/http"
)

func APIKeyAuth(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("X-API-Key")
            if !secureCompare(key, apiKey) {
                writeUnauthorized(w)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// secureCompare performs timing-safe comparison of API keys.
// Returns true only if both keys are equal and non-empty.
func secureCompare(provided, expected string) bool {
    // Length check for early exit (safe - no timing leak on length mismatch)
    if len(provided) != len(expected) {
        return false
    }
    // Reject empty keys
    if expected == "" {
        return false
    }
    // Constant-time comparison
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// writeUnauthorized returns a generic error that doesn't leak information.
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

## Verification

Run existing tests to ensure no regressions:
```bash
go test ./internal/api/middleware/... -v
```

## Dependencies

None - this plan is independent.

## Rollback

If issues arise, revert to the original comparison:
```go
if key != apiKey {
    writeUnauthorized(w, "invalid api key")
}
```