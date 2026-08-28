package api

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/pplmx/aurora/internal/api/handler"
	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/metrics"
)

type Server struct {
	db                   *sql.DB
	metrics              *metrics.Registry
	metricsOnce          sync.Once
	oracleRepo           oracle.Repository
	oracleMu             sync.Mutex
	oracleScheduler      *oracleapp.Scheduler
	oracleSchedulerFetch *oracleapp.FetchDataUseCase
	lotteryHandler       *handler.LotteryHandler
	votingHandler        *handler.VotingHandler
	nftHandler           *handler.NFTHandler
	tokenHandler         *handler.TokenHandler
	oracleHandler        *handler.OracleHandler
	blockchainHandler    *handler.BlockchainHandler
	eventStore           *infraevents.SQLiteEventStore

	// closers releases every SQLite handle NewServer opened (repos, event
	// store, replay protection) plus the shared blockchain DB. Tests use
	// Close() before t.TempDir() cleanup so Windows can delete the temp dir;
	// a long-lived server gets graceful handle release on shutdown.
	closers []func() error
}

// addCloser appends a handle to be released by Close(). nil-able to keep the
// error-path code terse.
func (s *Server) addCloser(c interface{ Close() error }) {
	s.closers = append(s.closers, c.Close)
}

// Close releases every SQLite handle the server opened. Idempotent-safe for
// callers; safe to call once at shutdown.
func (s *Server) Close() error {
	var firstErr error
	for _, c := range s.closers {
		if err := c(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := blockchain.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	s.closers = nil
	return firstErr
}

func NewServer() (*Server, error) {
	dbPath := blockchain.DBPath()
	db, err := blockchain.InitDB()
	if err != nil {
		return nil, err
	}

	srv := &Server{db: db, metrics: metrics.NewRegistry()}

	lotteryRepo, err := sqlite.NewLotteryRepository(dbPath)
	if err != nil {
		return nil, err
	}
	srv.addCloser(lotteryRepo)

	votingRepo := sqlite.NewVotingRepository(db)

	nftRepo, err := sqlite.NewNFTRepository(dbPath)
	if err != nil {
		return nil, err
	}
	srv.addCloser(nftRepo)

	tokenRepo, err := sqlite.NewTokenRepository(dbPath)
	if err != nil {
		return nil, err
	}
	srv.addCloser(tokenRepo)

	eventStore, err := infraevents.NewSQLiteEventStore(dbPath)
	if err != nil {
		return nil, err
	}
	srv.addCloser(eventStore)
	srv.eventStore = eventStore

	eventReader := sqlite.NewTokenEventReader(eventStore)

	eventBus := infraevents.NewSyncEventBus()
	// The CLI wiring (app/wire.go) subscribes the audit + stats handlers to
	// the bus; the API server path never did, so every token audit event the
	// service publishes (mint/transfer/approve/burn/transfer_from) was
	// silently dropped and GET /api/v1/token/history always returned empty on
	// the HTTP server. Wire the same handlers here so the production path
	// persists audit events to the event store (v1.73, ISS-080).
	//
	// The audit handler is wired with the event store as its durable outbox
	// (TASK-119, ISS-111): a transiently-failing delivery is parked in
	// pending_events instead of being dropped, and StartAuditOutboxDrainer
	// (started by cmd/api/main.go like the oracle scheduler) retries it. This
	// heals the gap v1.82 only reported.
	eventBus.SubscribeAll(infraevents.NewAuditHandlerWithOutbox(eventStore, eventStore).Handle)
	eventBus.SubscribeAll(infraevents.NewStatsHandler().Handle)

	replay, err := infraevents.NewSQLiteReplayProtection(dbPath)
	if err != nil {
		return nil, err
	}
	srv.addCloser(replay)

	chain := blockchain.GetBlockChain()
	txManager := sqlite.NewTxManager(tokenRepo.GetDB())
	tokenService := token.NewService(tokenRepo, txManager, eventBus, eventReader, replay, chain)

	oracleRepo, err := sqlite.NewOracleRepository(dbPath)
	if err != nil {
		return nil, err
	}
	srv.addCloser(oracleRepo)

	srv.lotteryHandler = handler.NewLotteryHandler(lotteryRepo)
	srv.votingHandler = handler.NewVotingHandler(votingRepo, sqlite.NewTxManager(db))
	srv.votingHandler.SetChain(blockchain.GetBlockChain())
	srv.nftHandler = handler.NewNFTHandler(nftRepo, sqlite.NewTxManager(nftRepo.GetDB()))
	srv.tokenHandler = handler.NewTokenHandler(tokenService)
	srv.oracleRepo = oracleRepo
	srv.oracleHandler = handler.NewOracleHandler(oracleRepo)
	srv.blockchainHandler = handler.NewBlockchainHandler()
	// Record API-fetched oracle data on-chain, consistent with the scheduler
	// and the package's documented "record on-chain" intent. Without this,
	// REST fetch results came back with BlockHeight 0.
	srv.oracleHandler.SetChain(blockchain.GetBlockChain())
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
	if s.eventStore == nil {
		return func() {}
	}
	drainer := infraevents.NewOutboxDrainer(s.eventStore, nil)
	drainCtx, cancel := context.WithCancel(ctx)
	go drainer.Run(drainCtx)
	return cancel
}

func (s *Server) StartOracleScheduler(ctx context.Context, checkEvery time.Duration) (stop func()) {
	if s.oracleRepo == nil {
		return func() {}
	}
	fetch := oracleapp.NewFetchDataUseCase(s.oracleRepo)
	// Record scheduler-fetched data on-chain, consistent with the REST handler
	// and CLI paths (NewServer: srv.oracleHandler.SetChain(blockchain.
	// GetBlockChain())). Without SetChain the continuously running scheduler
	// path silently skipped chain.AddLotteryRecord and stored block_height=0
	// for every observation it persisted (TASK-097, ISS-090).
	fetch.SetChain(blockchain.GetBlockChain())
	runner := func(ctx context.Context, sourceID string) error {
		// ctx-aware fetch: an in-flight HTTP request is interrupted when the
		// scheduler is shut down instead of blocking up to the client timeout
		// while srv.Close() tears the DB pool down under it (TASK-134).
		_, err := fetch.ExecuteContext(ctx, &oracleapp.FetchDataRequest{SourceID: sourceID})
		return err
	}
	sched := oracleapp.NewScheduler(s.oracleRepo, runner, checkEvery, nil)
	s.oracleMu.Lock()
	s.oracleScheduler = sched
	s.oracleSchedulerFetch = fetch
	s.oracleMu.Unlock()
	// Expose the scheduler's fetch-health stats to the protected
	// GET /api/v1/oracle/health endpoint (v1.33), not just /metrics/oracle.
	if s.oracleHandler != nil {
		s.oracleHandler.SetStats(sched)
	}
	schedCtx, cancel := context.WithCancel(ctx)
	go sched.Run(schedCtx)
	return cancel
}

// OracleMetricsHandler returns an http.Handler exposing the oracle fetch
// scheduler's per-source feed-health stats in Prometheus text format on
// /metrics/oracle (v1.16). It is nil-safe: before the scheduler starts it
// returns empty output.
func (s *Server) OracleMetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		s.oracleMu.Lock()
		sch := s.oracleScheduler
		s.oracleMu.Unlock()
		if sch == nil {
			_, _ = io.WriteString(w, "")
			return
		}
		_, _ = io.WriteString(w, sch.PrometheusText())
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
