# Plan 08-01: Implement Liveness Endpoint `/healthz`

**Phase:** 8 - Operations & Health
**Requirements:** OPS-01
**Status:** Planned

## Tasks

### 1. Create health handler file
**File:** `internal/api/health.go` (new)

```go
package api

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
}

// LivenessHandler returns 200 if the HTTP server is responding.
// This is a simple liveness probe - no dependency checks.
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	setHealthHeaders(w)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

func setHealthHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
```

### 2. Verify router integration
**File:** `internal/api/router.go`

The liveness handler will be registered in Plan 08-03. This plan is the implementation-only step.

## Files to Modify

| File | Action |
|------|--------|
| `internal/api/health.go` | Create |

## Success Criteria

- [ ] `GET /healthz` returns `{"status":"ok"}` with HTTP 200
- [ ] No dependency checks (database, etc.) - pure liveness
- [ ] Response includes proper headers (Content-Type, Cache-Control, X-Content-Type-Options)

## Testing

```bash
# Manual test
curl -i http://localhost:8080/healthz
# Expected: HTTP/1.1 200 OK, {"status":"ok"}
```