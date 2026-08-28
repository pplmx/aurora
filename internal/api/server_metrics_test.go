package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"

	"github.com/pplmx/aurora/internal/metrics"
)

// TestServer_MetricsHandlerExportsLiveCounters proves the /metrics handler is
// exportable from the Server as a standalone, reusable mount AND that it
// reflects the same live registry the router records into (v1.14 "export the
// metrics handler for external scraping"). Without the shared registry, the
// exported handler would be empty.
func TestServer_MetricsHandlerExportsLiveCounters(t *testing.T) {
	resetForAPITest(t)
	viper.Set("api.key", "k")
	db := openInMemorySQLite(t)
	srv := &Server{db: db, metrics: metrics.NewRegistry()}
	router := srv.Router()

	// Drive the router so the shared registry records traffic.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	}

	rec := httptest.NewRecorder()
	srv.MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("MetricsHandler status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "http_requests_total 2") {
		t.Errorf("exported metrics did not reflect 2 requests:\n%s", body)
	}
	if !strings.Contains(body, `http_requests_by_module{module="health"} 2`) {
		t.Errorf("expected health module bucket to show 2:\n%s", body)
	}
}

// TestServer_MetricsRegistry_ConcurrentLazyInit is the ISS-131 regression: the
// lazy init in MetricsRegistry was check-then-set with no lock, so concurrent
// calls on a Server built without the constructor (metrics == nil) could create
// two registries, silently splitting the request counters between them. Under
// -race the old code faults; the once-guarded init must hand every caller the
// same registry.
func TestServer_MetricsRegistry_ConcurrentLazyInit(t *testing.T) {
	srv := &Server{} // built without NewServer → metrics is nil

	const n = 64
	regs := make([]*metrics.Registry, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			regs[i] = srv.MetricsRegistry()
		}(i)
	}
	wg.Wait()

	for i := 1; i < n; i++ {
		if regs[i] != regs[0] {
			t.Fatalf("MetricsRegistry returned different instances: %p vs %p (counters would be split)", regs[i], regs[0])
		}
	}
	if regs[0] == nil {
		t.Fatal("MetricsRegistry must lazily create a registry")
	}
}
