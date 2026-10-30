// Package metrics provides a minimal, concurrency-safe request-observability
// registry and a plain-text exposition endpoint (Prometheus text format,
// stdlib-only). It fulfills the observability item deferred in the project
// plan without pulling an external metrics dependency or a framework change.
package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Middleware returns a chi-compatible middleware that records each request's
// status + latency into reg.
func Middleware(reg *Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			defer func() {
				// Observe aborted responses too. Recovery deliberately re-panics
				// with ErrAbortHandler after headers are committed; without a
				// deferred observation that request would disappear from the
				// metrics even though the client received part of it.
				reg.Observe(ww.status, time.Since(start), classifyModule(r.URL.Path))
			}()
			next.ServeHTTP(ww, r)
		})
	}
}

// knownAPIModules is the fixed set of route roots registered in
// api.newRouter under /api/v1/. classifyModule may emit a label only from this
// set; anything else falls into the single "other" bucket. Previously the raw
// third path segment was used verbatim, so an unauthenticated client could
// request /api/v1/<attacker-chosen>-{i}/... and grow distinct module labels
// (and /metrics payload) without bound — a remote memory-exhaustion path,
// since the metrics middleware is outermost, before auth (ISS-081).
var knownAPIModules = map[string]struct{}{
	"token": {}, "nft": {}, "voting": {}, "lottery": {}, "oracle": {}, "blockchain": {},
}

// maxMetricModuleLength bounds hashing for the untrusted path segment that is
// considered as a module label. Anything longer can only fall into "other", so
// there is no reason to scan or hash an attacker-sized segment.
const maxMetricModuleLength = 64

// classifyModule buckets a request path into a small set of high-level labels
// for the per-module metrics. API routes live under /api/v1/<module> (token,
// nft, voting, lottery, oracle, blockchain); health and metrics endpoints are
// separate buckets. Grouping by module (rather than full
// route pattern) keeps the label cardinality bounded while still showing which
// subsystem a request touched.
func classifyModule(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/"):
		rest := path[len("/api/v1/"):]
		module, _, _ := strings.Cut(rest, "/")
		if module == "" {
			return "api"
		}
		if len(module) > maxMetricModuleLength {
			return "other"
		}
		if _, ok := knownAPIModules[module]; ok {
			return module
		}
		return "other"
	case path == "/" || strings.HasPrefix(path, "/health") || strings.HasPrefix(path, "/readyz"):
		return "health"
	case strings.HasPrefix(path, "/metrics"):
		return "metrics"
	default:
		return "static"
	}
}

// statusWriter captures the response status code for metrics, defaulting to 200
// when the handler never calls WriteHeader.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusWriter) WriteHeader(code int) {
	if s.wroteHeader {
		return
	}
	s.wroteHeader = true
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(body []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(body)
}

// Unwrap lets http.ResponseController reach the underlying writer, so handlers
// can still Flush/Hijack/Push even while the contact ResponseWriter is this
// wrapper. This is the mechanism the net/http docs recommend for
// ResponseController-style access through middlewares.
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// Flush implements http.Flusher so streaming handlers (SSE, chunked or
// long-poll responses) keep working through the middleware. The metrics
// middleware is the innermost wrapper around every handler, so without these
// optional interfaces a handler that flushes would panic on an unchecked
// http.Flusher assertion or silently fail to flush via http.ResponseController.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		// Flush commits the implicit 200 even when the handler never called
		// WriteHeader. Later status writes are ignored by net/http and must
		// not replace what the client actually received in the metrics.
		s.wroteHeader = true
		f.Flush()
	}
}

// Hijack supports websocket-style upgrades through the middleware.
func (s *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := s.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("metrics statusWriter: underlying ResponseWriter does not implement http.Hijacker")
}

// Push implements http.Pusher (HTTP/2 server push) through the middleware.
func (s *statusWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := s.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// HistogramBuckets are latency buckets in seconds for the request-duration
// histogram (mirrors a small set of common Prometheus defaults).
var HistogramBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Registry accumulates HTTP request metrics. It is safe for concurrent use.
type Registry struct {
	mu           sync.RWMutex
	total        uint64
	status       map[int]uint64
	moduleTotal  map[string]uint64
	moduleStatus map[string]map[int]uint64
	buckets      map[int]uint64 // bucket index -> cumulative count
	sum          float64
	lastLatency  float64
}

func NewRegistry() *Registry {
	return &Registry{
		status:       make(map[int]uint64),
		moduleTotal:  make(map[string]uint64),
		moduleStatus: make(map[string]map[int]uint64),
		buckets:      make(map[int]uint64, len(HistogramBuckets)),
	}
}

// Observe records one request: its status code, latency in seconds, and the
// module label it belongs to (see classifyModule). A request with an empty
// module is bucketed under "unknown".
func (r *Registry) Observe(statusCode int, latency time.Duration, module string) {
	if module == "" {
		module = "unknown"
	}
	d := latency.Seconds()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.total++
	r.status[statusCode]++
	r.moduleTotal[module]++
	if r.moduleStatus[module] == nil {
		r.moduleStatus[module] = make(map[int]uint64)
	}
	r.moduleStatus[module][statusCode]++
	r.sum += d
	r.lastLatency = d
	for i, b := range HistogramBuckets {
		if d <= b {
			r.buckets[i]++
		}
	}
}

// Handler returns an http.Handler serving the current metrics in Prometheus
// text exposition format.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = io.WriteString(w, r.Expose())
	})
}

// Expose renders the metrics registry as Prometheus text format.
func (r *Registry) Expose() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var b strings.Builder
	fmt.Fprintln(&b, "# HELP aurora_http_requests_total Total HTTP requests handled.")
	fmt.Fprintln(&b, "# TYPE aurora_http_requests_total counter")
	fmt.Fprintf(&b, "aurora_http_requests_total %d\n", r.total)
	fmt.Fprintln(&b, "# HELP aurora_http_requests_by_status_total HTTP requests by response status code.")
	fmt.Fprintln(&b, "# TYPE aurora_http_requests_by_status_total counter")

	statuses := make([]int, 0, len(r.status))
	for s := range r.status {
		statuses = append(statuses, s)
	}
	sort.Ints(statuses)
	for _, s := range statuses {
		fmt.Fprintf(&b, "aurora_http_requests_by_status_total{code=\"%d\"} %d\n", s, r.status[s])
	}

	fmt.Fprintln(&b, "# HELP aurora_http_requests_by_module_total HTTP requests by module.")
	fmt.Fprintln(&b, "# TYPE aurora_http_requests_by_module_total counter")
	fmt.Fprintln(&b, "# HELP aurora_http_requests_by_module_status_total HTTP requests by module and response status code.")
	fmt.Fprintln(&b, "# TYPE aurora_http_requests_by_module_status_total counter")
	modules := make([]string, 0, len(r.moduleTotal))
	for m := range r.moduleTotal {
		modules = append(modules, m)
	}
	sort.Strings(modules)
	for _, m := range modules {
		fmt.Fprintf(&b, "aurora_http_requests_by_module_total{module=\"%s\"} %d\n", m, r.moduleTotal[m])

		st := r.moduleStatus[m]
		codes := make([]int, 0, len(st))
		for c := range st {
			codes = append(codes, c)
		}
		sort.Ints(codes)
		for _, c := range codes {
			fmt.Fprintf(&b, "aurora_http_requests_by_module_status_total{module=\"%s\",code=\"%d\"} %d\n", m, c, st[c])
		}
	}

	fmt.Fprintln(&b, "# HELP aurora_http_request_duration_seconds HTTP request latency in seconds.")
	fmt.Fprintln(&b, "# TYPE aurora_http_request_duration_seconds histogram")
	for i, lim := range HistogramBuckets {
		fmt.Fprintf(&b, "aurora_http_request_duration_seconds_bucket{le=\"%s\"} %d\n",
			strconv.FormatFloat(lim, 'f', -1, 64), r.buckets[i])
	}
	fmt.Fprintf(&b, "aurora_http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", r.total)
	fmt.Fprintf(&b, "aurora_http_request_duration_seconds_sum %s\n",
		strconv.FormatFloat(r.sum, 'f', -1, 64))
	fmt.Fprintf(&b, "aurora_http_request_duration_seconds_count %d\n", r.total)
	return b.String()
}
