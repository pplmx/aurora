package api

import (
	"context"
	"database/sql"
	"net/http"
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
	db             *sql.DB
	metrics        *metrics.Registry
	oracleRepo     oracle.Repository
	lotteryHandler *handler.LotteryHandler
	votingHandler  *handler.VotingHandler
	nftHandler     *handler.NFTHandler
	tokenHandler   *handler.TokenHandler
	oracleHandler  *handler.OracleHandler

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

	eventReader := sqlite.NewTokenEventReader(eventStore)

	eventBus := infraevents.NewSyncEventBus()

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
	srv.nftHandler = handler.NewNFTHandler(nftRepo, sqlite.NewTxManager(nftRepo.GetDB()))
	srv.tokenHandler = handler.NewTokenHandler(tokenService)
	srv.oracleRepo = oracleRepo
	srv.oracleHandler = handler.NewOracleHandler(oracleRepo)
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
func (s *Server) StartOracleScheduler(ctx context.Context, checkEvery time.Duration) (stop func()) {
	if s.oracleRepo == nil {
		return func() {}
	}
	fetch := oracleapp.NewFetchDataUseCase(s.oracleRepo)
	runner := func(sourceID string) error {
		_, err := fetch.Execute(&oracleapp.FetchDataRequest{SourceID: sourceID})
		return err
	}
	schedCtx, cancel := context.WithCancel(ctx)
	go oracleapp.NewScheduler(s.oracleRepo, runner, checkEvery, nil).Run(schedCtx)
	return cancel
}

// MetricsRegistry returns the request-metrics registry backing the in-process
// /metrics route. It is created lazily so a Server constructed without one
// (e.g. in tests that only build the router) still works; the produced/mounted
// Server owns a registry from NewServer.
func (s *Server) MetricsRegistry() *metrics.Registry {
	if s.metrics == nil {
		s.metrics = metrics.NewRegistry()
	}
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
