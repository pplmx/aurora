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
	reg.Observe(http.StatusOK, 5*time.Millisecond, "token")
	reg.Observe(http.StatusOK, 20*time.Millisecond, "token")
	reg.Observe(http.StatusNotFound, 2*time.Second, "nft")

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
	reg.Observe(http.StatusOK, time.Millisecond, "health")

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
func TestClassifyModule(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/api/v1/token/transfer", "token"},
		{"/api/v1/nft/mint", "nft"},
		{"/api/v1/voting/vote", "voting"},
		{"/api/v1/lottery/create", "lottery"},
		{"/api/v1/oracle/fetch", "oracle"},
		{"/api/v1/unknown/x", "unknown"},
		{"/healthz", "health"},
		{"/readyz", "health"},
		{"/health", "health"},
		{"/", "health"},
		{"/metrics", "metrics"},
		{"/index.html", "static"},
		{"/app.js", "static"},
	}
	for _, c := range cases {
		if got := classifyModule(c.path); got != c.want {
			t.Errorf("classifyModule(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestRegistry_PerModuleExpose(t *testing.T) {
	reg := NewRegistry()
	reg.Observe(http.StatusOK, time.Millisecond, "token")
	reg.Observe(http.StatusTeapot, time.Millisecond, "token")
	reg.Observe(http.StatusOK, time.Millisecond, "nft")

	out := reg.Expose()
	if !strings.Contains(out, `http_requests_by_module{module="token"} 2`) {
		t.Errorf("expected token module total 2 in:\n%s", out)
	}
	if !strings.Contains(out, `http_requests_by_module{module="nft"} 1`) {
		t.Errorf("expected nft module total 1 in:\n%s", out)
	}
	if !strings.Contains(out, `http_requests_by_module_status{module="token",code="418"} 1`) {
		t.Errorf("expected token 418 module-status 1 in:\n%s", out)
	}
}

func TestMiddleware_RecordsModule(t *testing.T) {
	reg := NewRegistry()
	handler := Middleware(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/transfer", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if reg.moduleTotal["token"] != 1 {
		t.Errorf("expected token module total 1, got %d", reg.moduleTotal["token"])
	}
	if reg.moduleStatus["token"][http.StatusOK] != 1 {
		t.Errorf("expected token 200 module-status 1")
	}
}
