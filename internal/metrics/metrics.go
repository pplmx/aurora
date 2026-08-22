// Package metrics provides a minimal, concurrency-safe request-observability
// registry and a plain-text exposition endpoint (Prometheus text format,
// stdlib-only). It fulfills the observability item deferred in the project
// plan without pulling an external metrics dependency or a framework change.
package metrics

import (
	"fmt"
	"io"
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
			reg.Observe(ww.status, time.Since(start))
		})
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

// HistogramBuckets are latency buckets in seconds for the request-duration
// histogram (mirrors a small set of common Prometheus defaults).
var HistogramBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Registry accumulates HTTP request metrics. It is safe for concurrent use.
type Registry struct {
	mu          sync.RWMutex
	total       uint64
	status      map[int]uint64
	buckets     map[int]uint64 // bucket index -> cumulative count
	sum         float64
	lastLatency float64
}

func NewRegistry() *Registry {
	return &Registry{
		status:  make(map[int]uint64),
		buckets: make(map[int]uint64, len(HistogramBuckets)),
	}
}

// Observe records one request: its status code and latency in seconds.
func (r *Registry) Observe(statusCode int, latency time.Duration) {
	d := latency.Seconds()
	r.mu.Lock()
	r.total++
	r.status[statusCode]++
	r.sum += d
	r.lastLatency = d
	for i, b := range HistogramBuckets {
		if d <= b {
			r.buckets[i]++
		}
	}
	r.mu.Unlock()
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
