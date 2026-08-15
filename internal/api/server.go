package api

import (
	"database/sql"
	"net/http"

	"github.com/pplmx/aurora/internal/api/handler"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type Server struct {
	db             *sql.DB
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

	srv := &Server{db: db}

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
	srv.oracleHandler = handler.NewOracleHandler(oracleRepo)
	return srv, nil
}

func (s *Server) Router() http.Handler {
	return newRouter(s)
}
