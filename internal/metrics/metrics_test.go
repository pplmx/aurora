package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegistry_ObserveAndExpose(t *testing.T) {
	reg := NewRegistry()
	reg.Observe(http.StatusOK, 5*time.Millisecond)
	reg.Observe(http.StatusOK, 20*time.Millisecond)
	reg.Observe(http.StatusNotFound, 2*time.Second)

	out := reg.Expose()
	if !strings.Contains(out, "http_requests_total 3") {
		t.Errorf("expected total 3, got:\n%s", out)
	}
	if !strings.Contains(out, `http_requests_by_status{code="200"} 2`) {
		t.Errorf("expected 200=2, got:\n%s", out)
	}
	if !strings.Contains(out, `http_requests_by_status{code="404"} 1`) {
		t.Errorf("expected 404=1, got:\n%s", out)
	}
	if !strings.Contains(out, "http_request_duration_seconds_count 3") {
		t.Errorf("expected duration count 3, got:\n%s", out)
	}
	// 2ms < 5ms bucket and 20ms falls between 10-25ms; the +Inf bucket carries
	// the total.
}

func TestRegistry_HandlerServesMetrics(t *testing.T) {
	reg := NewRegistry()
	reg.Observe(http.StatusOK, time.Millisecond)

	h := reg.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("unexpected content-type: %q", rr.Header().Get("Content-Type"))
	}
	if !strings.Contains(rr.Body.String(), "http_requests_total") {
		t.Errorf("expected metrics in body")
	}
}

func TestMiddleware_RecordsStatus(t *testing.T) {
	reg := NewRegistry()
	handler := Middleware(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if reg.total != 1 {
		t.Fatalf("expected 1 recorded request, got %d", reg.total)
	}
	if reg.status[http.StatusTeapot] != 1 {
		t.Errorf("expected one 418 status, got %d", reg.status[http.StatusTeapot])
	}
}
