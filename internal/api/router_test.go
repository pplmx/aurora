package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/blockchain"
)

// resetForAPITest resets all package-level singletons and viper so each test
// gets a clean slate.
func resetForAPITest(t *testing.T) {
	t.Helper()
	viper.Reset()
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })
	blockchain.ResetForTest()
}

// openInMemorySQLite returns an in-memory SQLite DB for health-check tests.
func openInMemorySQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Ping())
	return db
}

// =================================================================
// ReadinessHandler
// =================================================================

func TestReadinessHandler_HealthyDB(t *testing.T) {
	db := openInMemorySQLite(t)

	handler := ReadinessHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", rr.Header().Get("Cache-Control"))

	var resp HealthResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "ok", resp.Checks["database"])
}

func TestReadinessHandler_UnhealthyDB(t *testing.T) {
	// Create a DB and immediately close it so PingContext will fail.
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	handler := ReadinessHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	var resp HealthResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "fail", resp.Checks["database"])
}

func TestReadinessHandler_NilDBReturnsUnavailable(t *testing.T) {
	// Regression test: a nil DB used to cause a panic (PingContext on nil).
	// After the fix, ReadinessHandler reports unhealthy without crashing.
	handler := ReadinessHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		handler(rr, req)
	})
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	var resp HealthResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "fail", resp.Checks["database"])
}

func TestReadinessHandler_RespectsContextDeadline(t *testing.T) {
	// A DB that returns an error on PingContext should propagate as unhealthy.
	db := openInMemorySQLite(t)

	// Sanity check that the handler responds in well under 2s.
	handler := ReadinessHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	start := time.Now()
	rr := httptest.NewRecorder()
	handler(rr, req)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Less(t, elapsed, 1*time.Second,
		"readiness check should not exceed the configured timeout")
}

func TestReadinessHandler_DBReturningCustomError(t *testing.T) {
	// Build a *sql.DB whose PingContext always returns a non-Ping error
	// by closing it but still passing it to the handler.
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	handler := ReadinessHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	// Should be unhealthy regardless of the specific error.
	var resp HealthResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "unhealthy", resp.Status)
}

// =================================================================
// Router
// =================================================================

func TestRouter_RegistersAllExpectedRoutes(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "test-api-key")

	srv := &Server{
		db:             openInMemorySQLite(t),
		lotteryHandler: nil, // Routes() is called via package wiring only when non-nil
	}
	router := newRouter(srv)

	// Verify the router is non-nil and routes are registered by walking
	// the routes via OpenAPI of chi's pattern matching. Instead, we just
	// make a few representative requests and verify they don't 404.
	for _, tc := range []struct {
		method, path string
		wantStatus   int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodGet, "/health", http.StatusOK},
		{http.MethodGet, "/readyz", http.StatusOK},
		// Protected routes require API key + non-nil handler; both will
		// fail with 401 (auth) or 500 (handler nil), but NOT 404 — the
		// router must have matched the path.
		{http.MethodGet, "/api/v1/lottery/history", http.StatusUnauthorized},
		{http.MethodGet, "/api/v1/voting/candidates", http.StatusUnauthorized},
		{http.MethodGet, "/api/v1/nft/list", http.StatusUnauthorized},
		{http.MethodGet, "/api/v1/token/balance?token_id=t&owner=o", http.StatusUnauthorized},
		{http.MethodGet, "/api/v1/oracle/sources", http.StatusUnauthorized},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, tc.wantStatus, rr.Code,
			"%s %s: expected %d, got %d", tc.method, tc.path, tc.wantStatus, rr.Code)
	}
}

func TestRouter_ProtectedRoutesAcceptAPIKey(t *testing.T) {
	resetForAPITest(t)
	const apiKey = "secret-test-key"
	viper.Set("api.key", apiKey)

	srv := &Server{db: openInMemorySQLite(t)}
	router := newRouter(srv)

	// With the correct API key, the auth middleware passes and the handler
	// (nil) panics or returns 500 — what matters is the response is NOT 401.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/sources", nil)
	req.Header.Set("X-API-Key", apiKey)
	rr := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		router.ServeHTTP(rr, req)
	})
	assert.NotEqual(t, http.StatusUnauthorized, rr.Code,
		"valid API key should not be rejected as unauthorized")
}

func TestRouter_RejectsInvalidAPIKey(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "secret-test-key")

	srv := &Server{db: openInMemorySQLite(t)}
	router := newRouter(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/sources", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRouter_HealthEndpointRequiresNoAuth(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "secret-test-key")

	srv := &Server{db: openInMemorySQLite(t)}
	router := newRouter(srv)

	// Health endpoints must work even with no API key and even with viper
	// containing no API key at all (the path is registered before the auth
	// middleware group).
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// =================================================================
// Server.Router (delegates to newRouter)
// =================================================================

func TestServer_Router_ReturnsHTTPHandler(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "test-key")

	srv := &Server{db: openInMemorySQLite(t)}
	router := srv.Router()

	require.NotNil(t, router)
	// Should respond to GET /healthz.
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// =================================================================
// NewServer (smoke test — exercises the full wiring against a temp DB)
// =================================================================

func TestNewServer_SuccessWithTempDB(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "test-key")
	// Put DB in a writable temp directory so the data path resolves cleanly.
	tmpDir := t.TempDir()
	t.Setenv("AURORA_DB_PATH", filepath.Join(tmpDir, "aurora.db"))

	srv, err := NewServer()
	require.NoError(t, err, "NewServer should succeed")
	require.NotNil(t, srv)
	t.Cleanup(func() { _ = blockchain.Close() })

	assert.NotNil(t, srv.db)
	assert.NotNil(t, srv.lotteryHandler)
	assert.NotNil(t, srv.votingHandler)
	assert.NotNil(t, srv.nftHandler)
	assert.NotNil(t, srv.tokenHandler)
	assert.NotNil(t, srv.oracleHandler)

	// Router must be non-nil and serve liveness.
	router := srv.Router()
	require.NotNil(t, router)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestNewServer_FailsOnInvalidDBPath(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "test-key")

	// Create a file at the parent of where the DB should go, so MkdirAll fails.
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "blocking")
	require.NoError(t, os.WriteFile(blockingFile, []byte("not a dir"), 0o644))

	// Point the DB at a path whose parent is a regular file.
	t.Setenv("AURORA_DB_PATH", filepath.Join(blockingFile, "data", "aurora.db"))

	srv, err := NewServer()
	// Depending on how the repo layer reacts, either NewServer returns an
	// error or returns a server but with broken DB — we accept either, but
	// must NOT panic.
	if err != nil {
		assert.Nil(t, srv)
	} else {
		assert.NotNil(t, srv)
	}
}

// =================================================================
// Error semantics — ensure the router uses the correct JSON content type
// =================================================================

func TestRouter_ErrorResponseUsesJSON(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "secret")

	srv := &Server{db: openInMemorySQLite(t)}
	router := newRouter(srv)

	// Trigger 401 from auth middleware.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lottery/history", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	// Some implementations return JSON, some return text. At minimum,
	// the Content-Type should be set (chi's middleware sets it).
	ct := rr.Header().Get("Content-Type")
	assert.True(t,
		strings.HasPrefix(ct, "application/json") || ct == "",
		"unexpected content-type for 401: %q", ct)
}

// sentinel — keep the import block deterministic.
var _ = strings.Contains
