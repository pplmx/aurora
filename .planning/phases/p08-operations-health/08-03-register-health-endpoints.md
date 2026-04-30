# Plan 08-03: Register Health Endpoints Before Auth Middleware

**Phase:** 8 - Operations & Health
**Requirements:** OPS-04
**Status:** Planned

## Tasks

### 1. Update router.go to add health endpoints
**File:** `internal/api/router.go`

Replace the existing `/health` endpoint section with:

```go
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimw.Logger)
	r.Use(apimw.Recovery)
	r.Use(apimw.CORS)

	// Health endpoints - no auth required (must be before auth middleware)
	r.Get("/healthz", LivenessHandler)
	r.Get("/readyz", ReadinessHandler(s.db))

	// Keep existing /health for backward compatibility
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		LivenessHandler(w, r)
	})

	apiKey := config.GetAPIKey()

	r.Group(func(api chi.Router) {
		api.Use(apimw.APIKeyAuth(apiKey))
        // ... existing routes remain unchanged
    })
```

## Files to Modify

| File | Action |
|------|--------|
| `internal/api/router.go` | Add /healthz and /readyz routes before auth Group |

## Success Criteria

- [ ] `/healthz` accessible without API key (returns 200)
- [ ] `/readyz` accessible without API key (returns 200 when DB healthy)
- [ ] Existing `/health` still works (backward compatibility)
- [ ] All authenticated routes still require API key

## Verification

```bash
# Without API key - health endpoints should work
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/health

# With invalid API key - health endpoints should still work
curl -H "X-API-Key: invalid" http://localhost:8080/healthz
# Expected: 200 OK (not 401)

# With invalid API key - API endpoints should fail
curl -H "X-API-Key: invalid" http://localhost:8080/api/v1/lottery
# Expected: 401 Unauthorized
```