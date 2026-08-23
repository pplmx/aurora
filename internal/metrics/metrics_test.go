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

// TestMiddleware_StreamingInterfaces is a regression test for the metrics
// middleware not exposing the optional ResponseWriter interfaces. The metrics
// wrapper is the innermost middleware around every handler, so a handler that
// streams (SSE), upgrades (websocket) or pushes (HTTP/2) would otherwise hit a
// panic on an unchecked http.Flusher assertion or silently fail to flush via
// http.ResponseController. httptest.NewRecorder implements http.Flusher, so
// reaching it through the middleware proves the wrapper passes these through.
func TestMiddleware_StreamingInterfaces(t *testing.T) {
	reg := NewRegistry()
	flushed := false
	handler := Middleware(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("handler writer does not implement http.Flusher (streaming would break)")
		}
		f.Flush()
		flushed = true

		// http.ResponseController must also reach the underlying Flusher via
		// Unwrap.
		rc := http.NewResponseController(w)
		if err := rc.Flush(); err != nil {
			t.Errorf("ResponseController.Flush failed: %v", err)
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !flushed {
		t.Fatal("flush was never reached")
	}
	if reg.total != 1 {
		t.Fatalf("expected 1 recorded request, got %d", reg.total)
	}
}
