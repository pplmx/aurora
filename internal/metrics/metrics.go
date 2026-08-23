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
			next.ServeHTTP(ww, r)
			reg.Observe(ww.status, time.Since(start), classifyModule(r.URL.Path))
		})
	}
}

// classifyModule buckets a request path into a small set of high-level labels
// for the per-module metrics. API routes live under /api/v1/<module> (token,
// nft, voting, lottery, oracle); health and metrics endpoints and the static
// web UI are separate buckets. Grouping by module (rather than full route
// pattern) keeps the label cardinality bounded while still showing which
// subsystem a request touched.
func classifyModule(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/"):
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 3 && parts[2] != "" {
			return parts[2]
		}
		return "api"
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
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
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
	fmt.Fprintln(&b, "# TYPE http_requests_total counter")
	fmt.Fprintf(&b, "http_requests_total %d\n", r.total)
	fmt.Fprintln(&b, "# TYPE http_requests_by_status counter")

	statuses := make([]int, 0, len(r.status))
	for s := range r.status {
		statuses = append(statuses, s)
	}
	sort.Ints(statuses)
	for _, s := range statuses {
		fmt.Fprintf(&b, "http_requests_by_status{code=\"%d\"} %d\n", s, r.status[s])
	}

	fmt.Fprintln(&b, "# TYPE http_requests_by_module counter")
	modules := make([]string, 0, len(r.moduleTotal))
	for m := range r.moduleTotal {
		modules = append(modules, m)
	}
	sort.Strings(modules)
	for _, m := range modules {
		fmt.Fprintf(&b, "http_requests_by_module{module=\"%s\"} %d\n", m, r.moduleTotal[m])

		st := r.moduleStatus[m]
		codes := make([]int, 0, len(st))
		for c := range st {
			codes = append(codes, c)
		}
		sort.Ints(codes)
		for _, c := range codes {
			fmt.Fprintf(&b, "http_requests_by_module_status{module=\"%s\",code=\"%d\"} %d\n", m, c, st[c])
		}
	}

	fmt.Fprintln(&b, "# TYPE http_request_duration_seconds histogram")
	for i, lim := range HistogramBuckets {
		fmt.Fprintf(&b, "http_request_duration_seconds_bucket{le=\"%s\"} %d\n",
			strconv.FormatFloat(lim, 'f', -1, 64), r.buckets[i])
	}
	fmt.Fprintf(&b, "http_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", r.total)
	fmt.Fprintf(&b, "http_request_duration_seconds_sum %s\n",
		strconv.FormatFloat(r.sum, 'f', -1, 64))
	fmt.Fprintf(&b, "http_request_duration_seconds_count %d\n", r.total)
	return b.String()
}
