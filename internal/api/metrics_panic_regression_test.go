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
	r.Use(middleware.RealIP)
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
