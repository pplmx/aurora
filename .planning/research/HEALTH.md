# Health Check Patterns for Go Services with Chi Router

**Project:** Aurora v1.2 Operational Readiness
**Researched:** 2026-04-30
**Confidence:** HIGH (official Kubernetes docs + Chi middleware source + existing code analysis)

## Executive Summary

Aurora already has a basic `/health` endpoint in `internal/api/router.go`, but it doesn't distinguish between liveness and readiness semantics required for Kubernetes. This research recommends implementing proper `/healthz` (liveness) and `/readyz` (readiness) endpoints that bypass authentication, verify database connectivity for readiness, and ensure graceful shutdown with proper timeout handling.

## Key Findings

1. **Aurora's current `/health`** is a simple 200 OK with no checks - suitable for basic load balancer health but insufficient for k8s
2. **Kubernetes requires semantic separation**: liveness (`/healthz`) = "keep running?", readiness (`/readyz`) = "accept traffic?"
3. **Chi v5 provides `Heartbeat` middleware** for lightweight health endpoints, but custom handlers give more control
4. **Database connection check is critical** for readiness - Aurora's SQLite should be pinged
5. **Graceful shutdown needs timeout** - current `server.Close()` doesn't respect in-flight requests

## Kubernetes Health Check Semantics

### Liveness Probe (`/healthz` or `/live`)
- **Purpose:** Is the process alive and not deadlocked?
- **Failure action:** Kubernetes RESTARTS the container
- **Check:** Just return 200 if HTTP server responds
- **Anti-pattern:** Don't check dependencies here (will cause restart loops)

### Readiness Probe (`/readyz` or `/ready`)
- **Purpose:** Can the container accept traffic?
- **Failure action:** Kubernetes REMOVES from service endpoints (traffic stops)
- **Check:** Verify database connections, caches, external dependencies
- **Recovery:** Automatically returns to endpoints when checks pass

### Startup Probe
- **Purpose:** Slow-starting applications need extra time to initialize
- **Use when:** Application takes >10s to start
- **Aurora assessment:** Likely not needed (blockchain CLI tools start quickly)

### Recommended Endpoint Names

| Endpoint | Purpose | Auth Required | Checks |
|----------|---------|---------------|--------|
| `GET /healthz` | Liveness | No | Process alive |
| `GET /readyz` | Readiness | No | Database connected |
| `GET /health` | Keep existing | No | Simple OK (for load balancers) |

### Kubernetes Configuration Example

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 1
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 1
  failureThreshold: 3
```

## Chi Router Implementation Patterns

### Pattern 1: Heartbeat Middleware (Simple)

Chi v5 provides built-in `Heartbeat` middleware:

```go
import "github.com/go-chi/chi/v5/middleware"

func newRouter(s *Server) http.Handler {
    r := chi.NewRouter()
    
    // Heartbeat - bypasses auth, returns 200 immediately
    r.With(middleware.Heartbeat("/healthz")).Get("/", func(w http.ResponseWriter, r *http.Request) {})
    
    return r
}
```

**Pros:** Minimal code, uses Chi's built-in
**Cons:** Static response, no custom checks

### Pattern 2: Custom Health Handlers (Recommended)

```go
// internal/api/health.go
package api

import (
    "database/sql"
    "encoding/json"
    "net/http"
)

type HealthChecker interface {
    Ping() error
}

type HealthResponse struct {
    Status  string            `json:"status"`
    Checks  map[string]string `json:"checks,omitempty"`
}

// LivenessHandler - simple check, always returns 200 if server responds
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

// ReadinessHandler - checks dependencies, may return 503
func ReadinessHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        checks := make(map[string]string)
        
        // Check database connection
        if err := db.Ping(); err != nil {
            checks["database"] = "fail"
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusServiceUnavailable)
            _ = json.NewEncoder(w).Encode(HealthResponse{
                Status: "unhealthy",
                Checks: checks,
            })
            return
        }
        checks["database"] = "ok"
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _ = json.NewEncoder(w).Encode(HealthResponse{
            Status: "ok",
            Checks: checks,
        })
    }
}
```

### Pattern 3: Health Sub-router (Clean Organization)

```go
// internal/api/router.go
func newRouter(s *Server) http.Handler {
    r := chi.NewRouter()

    // Health endpoints - no auth required
    r.Route("/healthz", func(r chi.Router) {
        r.Use(middleware.NoCache)
        r.Get("/", LivenessHandler)
    })

    r.Route("/readyz", func(r chi.Router) {
        r.Use(middleware.NoCache)
        r.Get("/", ReadinessHandler(s.db))
    })

    // Authenticated API routes
    apiKey := config.GetAPIKey()
    r.Group(func(api chi.Router) {
        api.Use(apimw.APIKeyAuth(apiKey))
        // ... existing routes
    })

    return r
}
```

## Database Connection Checks

### SQLite with mattn/go-sqlite3

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

func ReadinessHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
        defer cancel()
        
        // Ping with timeout
        if err := db.PingContext(ctx); err != nil {
            writeUnhealthy(w, "database", err.Error())
            return
        }
        
        writeHealthy(w, map[string]string{"database": "ok"})
    }
}
```

### For future multi-database setups

```go
type ReadinessChecker struct {
    db      *sql.DB
    cache   *redis.Client  // if using Redis
    chain   blockchain.Chain
}

func (rc *ReadinessChecker) CheckAll() map[string]error {
    results := make(map[string]error)
    
    if rc.db != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        results["database"] = rc.db.PingContext(ctx)
    }
    
    // Add more checks as needed
    return results
}
```

## Graceful Shutdown Handling

### Current Implementation (cmd/api/main.go)

```go
// Current - problematic
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info().Msg("Shutting down server...")
if err := server.Close(); err != nil {
    // Close() is immediate, kills in-flight requests
}
```

### Recommended Implementation

```go
import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

const shutdownTimeout = 15 * time.Second

func main() {
    // ... server setup ...
    
    // Start server in goroutine
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal().Err(err).Msg("Server failed")
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    sig := <-quit

    logger.Info().Str("signal", sig.String()).Msg("Shutting down server...")

    // Create shutdown context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
    defer cancel()

    // Shutdown gracefully: stops accepting new connections, waits for existing
    if err := server.Shutdown(ctx); err != nil {
        logger.Error().Err(err).Msg("Server shutdown error")
        // Force close if graceful shutdown fails
        if err := server.Close(); err != nil {
            logger.Error().Err(err).Msg("Server force close error")
        }
    }

    logger.Info().Msg("Server stopped")
}
```

### Key Differences

| Approach | New Connections | In-Flight Requests | Timeout |
|----------|-----------------|-------------------|---------|
| `server.Close()` | Immediately rejected | Force closed | None |
| `server.Shutdown(ctx)` | Stop accepted | Wait for completion | Respects context |

### With Shutdown Hooks

For cleanup tasks (flushing buffers, closing DB pools):

```go
// Register cleanup on startup
server.RegisterOnShutdown(func() {
    logger.Info().Msg("Cleaning up resources...")
    // Close DB connections, flush caches, etc.
})

// Then on shutdown
if err := server.Shutdown(ctx); err != nil {
    // ...
}
```

## Health Endpoint Security

### Placement Matters

Health endpoints MUST be registered BEFORE auth middleware:

```go
// Correct order in router
r.Get("/healthz", LivenessHandler)        // No auth - check first
r.Get("/readyz", ReadinessHandler)        // No auth - check first

r.Group(func(api chi.Router) {
    api.Use(AuthMiddleware)               // Auth only for API routes
    // ...
})
```

### Headers for Health Responses

```go
func setHealthHeaders(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "no-store")
    w.Header().Set("X-Content-Type-Options", "nosniff")
}
```

## Recommended Implementation for Aurora

### File Structure

```
internal/api/
  health.go           # Health handlers
  health_test.go      # Unit tests
  router.go           # Updated with health routes
```

### internal/api/health.go

```go
package api

import (
    "context"
    "database/sql"
    "encoding/json"
    "net/http"
    "time"
)

type HealthResponse struct {
    Status  string            `json:"status"`
    Checks  map[string]string `json:"checks,omitempty"`
}

func LivenessHandler(w http.ResponseWriter, r *http.Request) {
    setHealthHeaders(w)
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

func ReadinessHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        setHealthHeaders(w)
        
        checks := make(map[string]string)
        ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
        defer cancel()
        
        if err := db.PingContext(ctx); err != nil {
            checks["database"] = "fail"
            w.WriteHeader(http.StatusServiceUnavailable)
            _ = json.NewEncoder(w).Encode(HealthResponse{
                Status: "unhealthy",
                Checks: checks,
            })
            return
        }
        checks["database"] = "ok"
        
        w.WriteHeader(http.StatusOK)
        _ = json.NewEncoder(w).Encode(HealthResponse{
            Status: "ok",
            Checks: checks,
        })
    }
}

func setHealthHeaders(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "no-store")
    w.Header().Set("X-Content-Type-Options", "nosniff")
}
```

### Updated router.go

```go
func newRouter(s *Server) http.Handler {
    r := chi.NewRouter()

    // Health endpoints (no auth - must be before auth middleware)
    r.Get("/healthz", LivenessHandler)
    r.Get("/readyz", ReadinessHandler(s.db))

    // Keep existing /health for backward compatibility
    r.Get("/health", LivenessHandler)

    apiKey := config.GetAPIKey()
    r.Group(func(api chi.Router) {
        api.Use(apimw.APIKeyAuth(apiKey))
        // ... existing routes
    })
    
    // ...
}
```

### Updated main.go

```go
const shutdownTimeout = 15 * time.Second

// In main(), replace server.Close() with:
ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    logger.Error().Err(err).Msg("Server shutdown error")
    if err := server.Close(); err != nil {
        logger.Error().Err(err).Msg("Server force close error")
    }
}
```

## Testing Recommendations

```go
// internal/api/health_test.go
func TestLivenessHandler(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
    w := httptest.NewRecorder()
    
    LivenessHandler(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "ok")
}

func TestReadinessHandler_DBHealthy(t *testing.T) {
    db, mock, _ := sqlmock.New()
    mock.ExpectPing()
    
    req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
    w := httptest.NewRecorder()
    
    ReadinessHandler(db)(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "ok")
}

func TestReadinessHandler_DBUnhealthy(t *testing.T) {
    db, mock, _ := sqlmock.New()
    mock.ExpectPing().WillReturnError(sql.ErrConnDone)
    
    req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
    w := httptest.NewRecorder()
    
    ReadinessHandler(db)(w, req)
    
    assert.Equal(t, http.StatusServiceUnavailable, w.Code)
    assert.Contains(t, w.Body.String(), "unhealthy")
}
```

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| K8s semantics | HIGH | Official Kubernetes docs verified |
| Chi patterns | HIGH | Chi v5 middleware source + existing code |
| Go graceful shutdown | HIGH | stdlib http.Server documentation |
| SQLite ping | MEDIUM | Standard approach, may need timeout tuning |
| Recommendations | HIGH | Follow established patterns |

## Sources

1. [Kubernetes Liveness/Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) - Official docs
2. [Chi v5 Middleware - Heartbeat](https://pkg.go.dev/github.com/go-chi/chi/middleware#Heartbeat) - GoDoc
3. [Go http.Server Shutdown](https://pkg.go.dev/net/http#Server.Shutdown) - Standard library docs
4. [Chi GitHub](https://github.com/go-chi/chi) - Router source and examples
5. Aurora codebase analysis - `internal/api/router.go`, `cmd/api/main.go`

## Open Questions

- Should `/readyz` check other dependencies (backup system, event store)?
- What's the appropriate timeout for SQLite ping (2s recommended)?
- Should health endpoints be exposed on separate port for k8s sidecar patterns?

---

*Research completed for Aurora v1.2 Operational Readiness milestone*
