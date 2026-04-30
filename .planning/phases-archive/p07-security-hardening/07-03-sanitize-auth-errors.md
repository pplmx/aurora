# Plan 07-03: Sanitize Auth Error Messages

**Phase:** 7 - Security Hardening
**Requirement:** SEC-03 (API auth error responses are generic to prevent information leakage)
**Status:** Not started

## Context

The current auth middleware returns `"invalid api key"` which reveals:
- That a key was provided (even if wrong format)
- That the server expects an API key
- Potentially hints about the validation logic

This information can help attackers understand the auth mechanism.

## Files to Modify

- `internal/api/middleware/auth.go`

## Tasks

### Task 1: Update writeUnauthorized signature
Change from `writeUnauthorized(w http.ResponseWriter, message string)` to `writeUnauthorized(w http.ResponseWriter)`

### Task 2: Remove message parameter
Use generic "authentication required" for all auth failures:
- Missing API key
- Invalid API key
- Malformed API key

### Task 3: Ensure no other error leakage
Check for any other places in auth.go that might leak information:
- Log key length (not value) server-side for debugging
- Don't reveal expected key format

## Expected Code Changes

### Before (auth.go - writeUnauthorized)

```go
func writeUnauthorized(w http.ResponseWriter, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnauthorized)
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "code":    "UNAUTHORIZED",
            "message": message,
        },
    })
}
```

### After (auth.go - writeUnauthorized)

```go
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

### Before (auth.go - APIKeyAuth middleware)

```go
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

### After (auth.go - APIKeyAuth middleware)

```go
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
```

## Verification

1. Test missing API key:
```bash
curl -i http://localhost:8080/api/v1/lottery
# Expected: 401 with {"error":{"code":"UNAUTHORIZED","message":"authentication required"}}
```

2. Test invalid API key:
```bash
curl -i -H "X-API-Key: wrong-key" http://localhost:8080/api/v1/lottery
# Expected: Same 401 response (not "invalid api key")
```

3. Test valid API key:
```bash
curl -i -H "X-API-Key: your-configured-key" http://localhost:8080/api/v1/lottery
# Expected: 200 (or appropriate response for your data)
```

4. Run tests:
```bash
go test ./internal/api/middleware/... -v
```

## Dependencies

Plan 07-01 (timing-safe comparison) should be completed first since this plan builds on the `secureCompare` function.

## Rollback

If issues arise, restore the parameterized writeUnauthorized:
```go
func writeUnauthorized(w http.ResponseWriter, message string) {
    // ... existing code ...
}
```
And update the call site.

## Security Note

This change makes all auth failures indistinguishable from the client perspective. This is intentional - attackers should not be able to:
1. Determine if their key format is correct
2. Determine if their key length is correct
3. Learn anything about the expected key structure

Server-side logging can capture details for debugging, but client responses are always generic.
