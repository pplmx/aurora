package api

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/pplmx/aurora/internal/app/container"
	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/metrics"
)

type Server struct {
	// db is kept for the router and tests that build a Server without the
	// full constructor; the container owns it in production.
	db          *sql.DB
	container   *container.Container
	metrics     *metrics.Registry
	metricsOnce sync.Once

	oracleRuntime *oracleapp.Runtime
	handlers      *handlers
}

// Close releases the container (every SQLite handle and the shared ledger).
// It is safe to call more than once; a server built without a container has
// nothing to release here.
func (s *Server) Close() error {
	if s.container == nil {
		return nil
	}
	return s.container.Close()
}

func NewServer() (*Server, error) {
	c, err := container.Open()
	if err != nil {
		return nil, err
	}
	srv := &Server{
		db:            c.DB,
		container:     c,
		metrics:       metrics.NewRegistry(),
		handlers:      newHandlers(c),
		oracleRuntime: oracleapp.NewRuntime(c.OracleRepo, c.OracleFetcher, c.Chain),
	}
	return srv, nil
}

// StartOracleScheduler starts a background goroutine that periodically fetches
// enabled oracle sources on their configured Interval, and returns a stop
// function that cancels it. checkEvery is the poll cadence (>= 1s; <=0 defaults
// to 1s). It is a no-op if the server has no oracle repository. The scheduler
// is deliberately not started by NewServer so tests stay hermetic; cmd/api/main
// starts it on boot.
// StartAuditOutboxDrainer starts the background retry loop that delivers audit
// events parked in pending_events (TASK-119, ISS-111). The audit handler wired
// in NewServer parks a transiently-failed delivery in the outbox instead of
// dropping it; this drainer retries each parked event until it lands. Like the
// oracle scheduler it is deliberately not started by NewServer so tests stay
// hermetic; cmd/api/main.go starts it on boot. It is a no-op if the server has
// no event store.
func (s *Server) StartAuditOutboxDrainer(ctx context.Context) (stop func()) {
	if s.container == nil {
		return func() {}
	}
	return s.container.StartOutboxDrainer(ctx)
}

// OracleHealthStats returns the running scheduler's per-source fetch-health
// counters, or nil before StartOracleScheduler has built it. The desktop
// window reads the same counters the protected /api/v1/oracle/health endpoint
// publishes, so a native page and the REST surface cannot disagree about
// whether a feed is failing.
func (s *Server) OracleHealthStats() []oracleapp.SourceStat {
	if s.oracleRuntime == nil {
		return nil
	}
	return s.oracleRuntime.Stats()
}

func (s *Server) StartOracleScheduler(ctx context.Context, checkEvery time.Duration) (stop func()) {
	if s.oracleRuntime == nil {
		return func() {}
	}
	stop = s.oracleRuntime.Start(ctx, checkEvery)
	// Expose the scheduler's fetch-health stats to the protected
	// GET /api/v1/oracle/health endpoint (v1.33), not just /metrics/oracle.
	if s.handlers != nil {
		s.handlers.oracle.SetStats(s.oracleRuntime)
	}
	return stop
}

// OracleMetricsHandler returns an http.Handler exposing the oracle fetch
// scheduler's per-source feed-health stats in Prometheus text format on
// /metrics/oracle (v1.16). It is nil-safe: before the scheduler starts it
// returns empty output.
func (s *Server) OracleMetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		if s.oracleRuntime == nil {
			_, _ = io.WriteString(w, "")
			return
		}
		_, _ = io.WriteString(w, s.oracleRuntime.PrometheusText())
	})
}

// MetricsRegistry returns the request-metrics registry backing the in-process
// /metrics route. It is created lazily so a Server constructed without one
// (e.g. in tests that only build the router) still works; the produced/mounted
// Server owns a registry from NewServer.
//
// The lazy init is once-guarded: the check-then-set was unsynchronized, so two
// concurrent calls on a Server built without the constructor could create two
// registries and split the request counters between them (ISS-131). sync.Once
// makes the first caller own creation and every subsequent caller observe it.
func (s *Server) MetricsRegistry() *metrics.Registry {
	s.metricsOnce.Do(func() {
		if s.metrics == nil {
			s.metrics = metrics.NewRegistry()
		}
	})
	return s.metrics
}

// MetricsHandler returns a standalone http.Handler exposing the same request
// metrics in Prometheus text format, independent of the router mount. This is
// the reusable surface operators can mount on a separate metrics port (e.g.
// k8s /lb scraping) without serving it on the public API router.
func (s *Server) MetricsHandler() http.Handler {
	return s.MetricsRegistry().Handler()
}

func (s *Server) Router() http.Handler {
	return newRouter(s)
}

// Services exposes the assembled application services so a non-HTTP
// presentation layer (the desktop app's native UI) can drive the same
// repositories and services the REST handlers use, without re-implementing
// the container's wiring or opening a second set of SQLite handles.
//
// It is a read-only view: the caller must not close these handles.
func (s *Server) Services() container.Services {
	return s.container
}
