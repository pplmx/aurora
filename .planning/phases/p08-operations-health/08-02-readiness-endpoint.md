# Plan 08-02: Implement Readiness Endpoint `/readyz`

**Phase:** 8 - Operations & Health
**Requirements:** OPS-02
**Status:** Planned

## Tasks

### 1. Add readiness handler to health.go
**File:** `internal/api/health.go`

Add the following to the existing health.go file (after LivenessHandler):

```go
import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// ReadinessHandler returns 200 if the database is reachable.
// Returns 503 Service Unavailable if database ping fails.
func ReadinessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setHealthHeaders(w)

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(HealthResponse{
				Status: "unhealthy",
				Checks: map[string]string{"database": "fail"},
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status: "ok",
			Checks: map[string]string{"database": "ok"},
		})
	}
}
```

Update the HealthResponse struct to support Checks field:

```go
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}
```

### 2. Update Router to pass db to health handlers
**File:** `internal/api/router.go` (handled in Plan 08-03)

## Files to Modify

| File | Action |
|------|--------|
| `internal/api/health.go` | Add ReadinessHandler function |

## Success Criteria

- [ ] `GET /readyz` returns 200 with `{"status":"ok","checks":{"database":"ok"}}` when DB is healthy
- [ ] `GET /readyz` returns 503 with `{"status":"unhealthy","checks":{"database":"fail"}}` when DB is unreachable
- [ ] Database ping has 2 second timeout
- [ ] Response includes proper headers

## Testing

```bash
# With healthy database
curl -i http://localhost:8080/readyz
# Expected: HTTP/1.1 200 OK, {"status":"ok","checks":{"database":"ok"}}

# With unreachable database (simulate by stopping DB)
# Expected: HTTP/1.1 503 Service Unavailable, {"status":"unhealthy","checks":{"database":"fail"}}
```
