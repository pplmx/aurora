package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
