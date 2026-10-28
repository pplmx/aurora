package container

import (
	"context"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/ledger"
)

// Services is the read-only dependency view a presentation layer needs. It
// deliberately exposes no database handle and no SQLite repository type.
var _ Services = (*Container)(nil)

type Services interface {
	LotteryRepository() lottery.Repository
	VotingRepository() voting.TransactableRepository
	NFTRepository() nft.TransactableRepository
	OracleRepository() oracle.Repository
	Token() token.Service
	TokenTransactions() voting.TransactionManager
	VotingTransactions() voting.TransactionManager
	NFTTransactions() nft.TransactionManager
	Ledger() Chain
	Fetcher() oracleapp.FetcherInterface
	Backup(ctx context.Context, dest string) (string, error)
}

// Chain is the ledger view shared by the HTTP and desktop adapters.
type Chain interface {
	AddBlock(data string) (int64, error)
	AddRecord(data string) (int64, error)
	VerifyIntegrity() *ledger.IntegrityReport
	Len() int
	BlockEntries(limit int) []ledger.BlockEntry
	BlockEntriesPage(limit, offset int) []ledger.BlockEntry
	BlockEntriesPageMatching(limit, offset int, match func(ledger.BlockEntry) bool) ([]ledger.BlockEntry, int)
}

func (c *Container) LotteryRepository() lottery.Repository           { return c.LotteryRepo }
func (c *Container) VotingRepository() voting.TransactableRepository { return c.VotingRepo }
func (c *Container) NFTRepository() nft.TransactableRepository       { return c.NFTRepo }
func (c *Container) OracleRepository() oracle.Repository             { return c.OracleRepo }
func (c *Container) Token() token.Service                            { return c.TokenService }
func (c *Container) TokenTransactions() voting.TransactionManager    { return c.TxManager }
func (c *Container) VotingTransactions() voting.TransactionManager   { return c.VotingTx }
func (c *Container) NFTTransactions() nft.TransactionManager         { return c.NFTTx }
func (c *Container) Ledger() Chain                                   { return c.Chain }
func (c *Container) Fetcher() oracleapp.FetcherInterface             { return c.OracleFetcher }
