package metrics

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var classifiedModule string

func TestRegistry_ObserveAndExpose(t *testing.T) {
	reg := NewRegistry()
	reg.Observe(http.StatusOK, 5*time.Millisecond, "token")
	reg.Observe(http.StatusOK, 20*time.Millisecond, "token")
	reg.Observe(http.StatusNotFound, 2*time.Second, "nft")

	out := reg.Expose()
	if !strings.Contains(out, "aurora_http_requests_total 3") {
		t.Errorf("expected total 3, got:\n%s", out)
	}
	if !strings.Contains(out, `aurora_http_requests_by_status_total{code="200"} 2`) {
		t.Errorf("expected 200=2, got:\n%s", out)
	}
	if !strings.Contains(out, `aurora_http_requests_by_status_total{code="404"} 1`) {
		t.Errorf("expected 404=1, got:\n%s", out)
	}
	if !strings.Contains(out, "aurora_http_request_duration_seconds_count 3") {
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
	if !strings.Contains(rr.Body.String(), "aurora_http_requests_total") {
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

// TestMiddleware_RecordsFirstCommittedStatus keeps metrics aligned with the
// status the client actually received. net/http ignores later WriteHeader
// calls once a response is committed, including an implicit 200 from Write or
// Flush; the metrics wrapper previously let a later call overwrite its status.
func TestMiddleware_RecordsFirstCommittedStatus(t *testing.T) {
	cases := []struct {
		name   string
		write  func(http.ResponseWriter)
		status int
	}{
		{
			name: "duplicate header",
			write: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
				w.WriteHeader(http.StatusInternalServerError)
			},
			status: http.StatusNoContent,
		},
		{
			name: "body before header",
			write: func(w http.ResponseWriter) {
				_, _ = w.Write([]byte("ok"))
				w.WriteHeader(http.StatusInternalServerError)
			},
			status: http.StatusOK,
		},
		{
			name: "flush before header",
			write: func(w http.ResponseWriter) {
				w.(http.Flusher).Flush()
				w.WriteHeader(http.StatusInternalServerError)
			},
			status: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := NewRegistry()
			handler := Middleware(reg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				tc.write(w)
			}))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, tc.status, rr.Code, "the recorder must model the wire status")
			require.Equal(t, uint64(1), reg.status[tc.status],
				"metrics must record the first committed status")
			require.NotContains(t, reg.status, http.StatusInternalServerError,
				"a status ignored by net/http must not appear in metrics")
		})
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

func TestMiddleware_RecordsAbortedResponse(t *testing.T) {
	reg := NewRegistry()
	handler := Middleware(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPartialContent)
		panic(http.ErrAbortHandler)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history", nil)
	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	require.Equal(t, uint64(1), reg.total,
		"an aborted response must still be counted")
	require.Equal(t, uint64(1), reg.status[http.StatusPartialContent],
		"metrics must record the status committed before the abort")
}

func TestClassifyModule(t *testing.T) {
	cases := []struct{ path, want string }{
		// Real registered route roots keep their own label.
		{"/api/v1/token/transfer", "token"},
		{"/api/v1/nft/mint", "nft"},
		{"/api/v1/voting/vote", "voting"},
		{"/api/v1/lottery/create", "lottery"},
		{"/api/v1/oracle/fetch", "oracle"},
		{"/api/v1/blockchain/info", "blockchain"},
		// Anything that is not one of the fixed route roots collapses into a
		// single "other" bucket (ISS-081): attacker-chosen path segments must
		// never grow the label set.
		{"/api/v1/unknown/x", "other"},
		{"/api/v1/smoke-token-nft/x", "other"},
		{"/api/v1/token_99/transfer", "other"},
		{"/api/v1/", "api"},
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

func TestClassifyModule_LongUnknownSegmentStaysBounded(t *testing.T) {
	longPath := "/api/v1/" + strings.Repeat("x", 1<<20)

	require.Equal(t, "other", classifyModule(longPath))
	allocs := testing.AllocsPerRun(100, func() {
		classifiedModule = classifyModule(longPath)
	})
	require.Zero(t, allocs,
		"metric classification must not materialize a slice for an attacker-sized path")
}

// TestRegistry_UnknownModuleLabelsStayBounded is the ISS-081 regression at the
// registry level: no matter how many distinct attacker-chosen paths are
// observed, the per-module maps only ever hold the known modules plus the
// "other" bucket, so memory and /metrics payload do not grow without bound.
func TestRegistry_UnknownModuleLabelsStayBounded(t *testing.T) {
	reg := NewRegistry()
	for i := 0; i < 1000; i++ {
		reg.Observe(http.StatusNotFound, time.Millisecond,
			classifyModule("/api/v1/arbitrary-"+strconv.Itoa(i)+"/max"))
	}

	var counted int
	out := reg.Expose()
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "aurora_http_requests_by_module_total{") {
			counted++
		}
	}
	// token, nft, voting, lottery, oracle, blockchain, api, health, metrics,
	// static, other + any status-only lines — module lines must stay tiny.
	require.Less(t, counted, 15, "distinct module labels must be bounded; /metrics payload grew: %d", counted)
}

func TestRegistry_PerModuleExpose(t *testing.T) {
	reg := NewRegistry()
	reg.Observe(http.StatusOK, time.Millisecond, "token")
	reg.Observe(http.StatusTeapot, time.Millisecond, "token")
	reg.Observe(http.StatusOK, time.Millisecond, "nft")

	out := reg.Expose()
	if !strings.Contains(out, `aurora_http_requests_by_module_total{module="token"} 2`) {
		t.Errorf("expected token module total 2 in:\n%s", out)
	}
	if !strings.Contains(out, `aurora_http_requests_by_module_total{module="nft"} 1`) {
		t.Errorf("expected nft module total 1 in:\n%s", out)
	}
	if !strings.Contains(out, `aurora_http_requests_by_module_status_total{module="token",code="418"} 1`) {
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

// TestExpositionFollowsPrometheusConventions pins the checks `promtool check
// metrics` performs: a family declares HELP and TYPE before its first sample,
// and a counter's name ends in _total. The exposition has no schema, so a
// missing HELP or a counter without the suffix otherwise reaches a dashboard
// unnoticed (CI has no promtool to catch it).
func TestExpositionFollowsPrometheusConventions(t *testing.T) {
	reg := NewRegistry()
	reg.Observe(http.StatusOK, 5*time.Millisecond, "token")
	reg.Observe(http.StatusNotFound, time.Second, "other")
	out := reg.Expose()

	help := make(map[string]bool)
	kind := make(map[string]string)
	seen := make(map[string]bool)

	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# HELP "):
			name := strings.Fields(line)[2]
			if seen[name] {
				t.Fatalf("HELP for %s comes after its samples:\n%s", name, out)
			}
			if help[name] {
				t.Fatalf("duplicate HELP for %s", name)
			}
			help[name] = true
		case strings.HasPrefix(line, "# TYPE "):
			fields := strings.Fields(line)
			name, typ := fields[2], fields[3]
			if seen[name] {
				t.Fatalf("TYPE for %s comes after its samples:\n%s", name, out)
			}
			kind[name] = typ
		case line == "" || strings.HasPrefix(line, "#"):
			continue
		default:
			name := line
			if i := strings.IndexByte(name, '{'); i >= 0 {
				name = name[:i]
			}
			name = strings.SplitN(name, " ", 2)[0]
			seen[name] = true

			family := name
			if base, ok := histogramFamily(name, kind); ok {
				family = base
			}
			if !help[family] {
				t.Fatalf("sample %s has no HELP line for %s:\n%s", name, family, out)
			}
			if kind[family] == "" {
				t.Fatalf("sample %s has no TYPE line for %s:\n%s", name, family, out)
			}
		}
	}

	for name, typ := range kind {
		if typ == "counter" && !strings.HasSuffix(name, "_total") {
			t.Errorf("counter %s must end in _total", name)
		}
	}
}

// histogramFamily maps a histogram's _bucket/_sum/_count sample back to the
// family its TYPE line declares.
func histogramFamily(sample string, kind map[string]string) (string, bool) {
	for _, suffix := range []string{"_bucket", "_sum", "_count"} {
		if base, ok := strings.CutSuffix(sample, suffix); ok && kind[base] == "histogram" {
			return base, true
		}
	}
	return "", false
}

// TestRegistryGauges pins the scrape-time values: a registered gauge renders
// with HELP/TYPE, a reader that reports false omits the line for that scrape,
// and readers run after the registry lock is released (a reader that observes
// under the lock would deadlock the test).
func TestRegistryGauges(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterGauge("aurora_outbox_pending_events", "Parked audit events.", func() (float64, bool) {
		reg.Observe(200, time.Millisecond, "lottery") // must not deadlock
		return 3, true
	})
	reg.RegisterGauge("aurora_unavailable_gauge", "No value right now.", func() (float64, bool) {
		return 0, false
	})

	out := reg.Expose()
	assert.Contains(t, out, "# HELP aurora_outbox_pending_events Parked audit events.\n")
	assert.Contains(t, out, "# TYPE aurora_outbox_pending_events gauge\n")
	assert.Contains(t, out, "\naurora_outbox_pending_events 3\n")
	assert.NotContains(t, out, "aurora_unavailable_gauge")
}

// TestConstantGaugeEscapesLabels pins the text-format escaping: a version string
// carrying a quote must not break the exposition.
func TestConstantGaugeEscapesLabels(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterConstantGauge("aurora_build_info", "Build identity.",
		map[string]string{"version": `1.0 "quoted"`, "go": "go1.27"}, 1)
	out := reg.Expose()
	assert.Contains(t, out, `aurora_build_info{go="go1.27",version="1.0 \"quoted\""} 1`)
}
