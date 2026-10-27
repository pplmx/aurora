package api

import (
	"github.com/pplmx/aurora/internal/api/handler"
	"github.com/pplmx/aurora/internal/app/container"
	"github.com/pplmx/aurora/internal/config"
)

// handlers is the HTTP surface's handler set. Keeping construction here makes
// NewServer a lifecycle owner (container, metrics, scheduler) instead of a
// second dependency-injection file.
type handlers struct {
	lottery    *handler.LotteryHandler
	voting     *handler.VotingHandler
	nft        *handler.NFTHandler
	token      *handler.TokenHandler
	oracle     *handler.OracleHandler
	blockchain *handler.BlockchainHandler
}

func newHandlers(c *container.Container) *handlers {
	voting := handler.NewVotingHandler(c.VotingRepo, c.VotingTx)
	voting.SetChain(c.Chain)
	oracle := handler.NewOracleHandler(c.OracleRepo)
	oracle.SetChain(c.Chain)
	return &handlers{
		lottery:    handler.NewLotteryHandler(c.LotteryRepo, config.LotteryDefaultCount()),
		voting:     voting,
		nft:        handler.NewNFTHandler(c.NFTRepo, c.NFTTx),
		token:      handler.NewTokenHandler(c.TokenService),
		oracle:     oracle,
		blockchain: handler.NewBlockchainHandler(),
	}
}
