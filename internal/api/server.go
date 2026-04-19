package api

import (
	"net/http"

	"github.com/pplmx/aurora/internal/api/handler"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/token"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type Server struct {
	lotteryHandler *handler.LotteryHandler
	votingHandler  *handler.VotingHandler
	nftHandler     *handler.NFTHandler
	tokenHandler   *handler.TokenHandler
	oracleHandler  *handler.OracleHandler
}

func NewServer() (*Server, error) {
	dbPath := blockchain.DBPath()
	db, err := blockchain.InitDB()
	if err != nil {
		return nil, err
	}

	lotteryRepo, err := sqlite.NewLotteryRepository(dbPath)
	if err != nil {
		return nil, err
	}

	votingRepo := sqlite.NewVotingRepository(db)

	nftRepo, err := sqlite.NewNFTRepository(dbPath)
	if err != nil {
		return nil, err
	}

	tokenRepo, err := sqlite.NewTokenRepository(dbPath)
	if err != nil {
		return nil, err
	}

	eventStore, err := infraevents.NewSQLiteEventStore(dbPath)
	if err != nil {
		return nil, err
	}

	eventReader := sqlite.NewTokenEventReader(eventStore)

	eventBus := infraevents.NewSyncEventBus()

	replay, err := infraevents.NewSQLiteReplayProtection(dbPath)
	if err != nil {
		return nil, err
	}

	chain := blockchain.GetBlockChain()
	tokenService := token.NewService(tokenRepo, eventBus, eventReader, replay, chain)

	oracleRepo, err := sqlite.NewOracleRepository(dbPath)
	if err != nil {
		return nil, err
	}

	return &Server{
		lotteryHandler: handler.NewLotteryHandler(lotteryRepo),
		votingHandler:  handler.NewVotingHandler(votingRepo),
		nftHandler:     handler.NewNFTHandler(nftRepo),
		tokenHandler:   handler.NewTokenHandler(tokenService),
		oracleHandler:  handler.NewOracleHandler(oracleRepo),
	}, nil
}

func (s *Server) Router() http.Handler {
	return newRouter(s)
}
