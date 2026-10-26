package deskapp

import (
	"github.com/pplmx/aurora/desktop/internal/nativeui"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
)

// nativeServices adapts the application layer to the view layer's Services
// interface.
//
// The adapter package maps domain entities into nativeui row types, so a
// domain change lands here instead of rippling through layout code, and the
// views never import a domain package. Each module keeps its methods in its
// own native_*.go file.
type nativeServices struct {
	lotteryRepo lottery.Repository
	votingRepo  voting.TransactableRepository
	nftRepo     nft.TransactableRepository
	oracleRepo  oracle.Repository
	chain       *blockchain.Chain

	// tokenService carries the audit bus and replay protection the token use
	// cases publish through, so a window mint is recorded exactly like a CLI
	// or REST one.
	tokenService token.Service
	txManager    voting.TransactionManager
	votingSvc    voting.Service
	nftTx        nft.TransactionManager
	// health reads the oracle scheduler's per-source counters. It is a
	// function rather than a snapshot because the scheduler is built after the
	// server and keeps counting while the window is open; nil means this
	// process has no scheduler (tests, or an embryo runtime).
	health func() []oracleapp.SourceStat
	// oracleFetcher is injected by the composition root; the adapter does not
	// import infra/http.
	oracleFetcher oracleapp.FetcherInterface
}

var _ nativeui.Services = (*nativeServices)(nil)

func newNativeServices(
	lotteryRepo lottery.Repository,
	votingRepo voting.TransactableRepository,
	nftRepo nft.TransactableRepository,
	oracleRepo oracle.Repository,
	tokenService token.Service,
	txManager voting.TransactionManager,
	nftTx nft.TransactionManager,
	chain *blockchain.Chain,
	oracleFetcher oracleapp.FetcherInterface,
	health func() []oracleapp.SourceStat,
) *nativeServices {
	return &nativeServices{
		lotteryRepo:   lotteryRepo,
		votingRepo:    votingRepo,
		nftRepo:       nftRepo,
		oracleRepo:    oracleRepo,
		chain:         chain,
		tokenService:  tokenService,
		txManager:     txManager,
		votingSvc:     voting.NewEd25519Service(),
		nftTx:         nftTx,
		oracleFetcher: oracleFetcher,
		health:        health,
	}
}

func joinCSV(items []string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += ","
		}
		out += it
	}
	return out
}
