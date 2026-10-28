package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	apimw "github.com/pplmx/aurora/internal/api/middleware"
	"github.com/pplmx/aurora/internal/metrics"
)

// TestMetrics_RecordsRecoveredPanicAs500 locks the v1.59 ordering fix: the
// metrics middleware must be registered OUTER to Recovery so a panicking
// handler is recorded as a status-500 request. It mirrors newRouter's use
// order (metrics first = outermost in chi). Before the fix metrics was
// registered last (innermost), so a panic unwound past it before Recovery
// caught it and the 500 was never counted (ISS-064, TASK-072).
func TestMetrics_RecordsRecoveredPanicAs500(t *testing.T) {
	resetForAPITest(t)

	reg := metrics.NewRegistry()
	r := chi.NewRouter()
	// metrics registered FIRST (outermost), matching the v1.59 router.
	r.Use(metrics.Middleware(reg))
	r.Use(middleware.RequestID)
	r.Use(apimw.PeerIP)
	r.Use(apimw.ClientIP(nil))
	r.Use(apimw.Logger)
	r.Use(apimw.Recovery)
	r.Use(apimw.CORS(nil))
	r.Use(apimw.SecurityHeaders)

	r.Get("/boom", func(w http.ResponseWriter, r *http.Request) {
		panic("intentional test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// The client still gets a 500 via Recovery.
	require.Equal(t, http.StatusInternalServerError, rr.Code)

	// And the request is now visible in the metrics: total incremented and the
	// recovered panic is attributed to status 500 (previously it was invisible).
	exp := reg.Expose()
	t.Logf("metrics:\n%s", exp)
	require.True(t, strings.Contains(exp, "http_requests_total 1"),
		"a recovered panic must be counted as a request")
	require.True(t, strings.Contains(exp, `code="500"`),
		"a recovered panic must be attributed to status 500")
}

// TestMetrics_RecordsCommittedFlushAs200 verifies that a panic after an
// explicit Flush does not rewrite the already-committed client status to 500
// in metrics. Recovery must leave the response alone because Flush sends the
// successful 200 headers before the panic.
func TestMetrics_RecordsCommittedFlushAs200(t *testing.T) {
	resetForAPITest(t)

	reg := metrics.NewRegistry()
	r := chi.NewRouter()
	r.Use(metrics.Middleware(reg))
	r.Use(apimw.Recovery)
	r.Get("/flush-then-panic", func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		panic("intentional panic after flush")
	})

	req := httptest.NewRequest(http.MethodGet, "/flush-then-panic", nil)
	rr := httptest.NewRecorder()
	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		r.ServeHTTP(rr, req)
	})

	require.Equal(t, http.StatusOK, rr.Code)
	require.Empty(t, rr.Body.String())

	exp := reg.Expose()
	require.Contains(t, exp, "http_requests_total 1")
	require.Contains(t, exp, `code="200"`,
		"metrics must report the status the client actually received")
	require.NotContains(t, exp, `code="500"`,
		"Recovery must not attempt a 500 after headers were flushed")
}
