package container

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	// PendingAuditEvents counts audit events parked in the outbox, the number
	// `aurora doctor` reports. The desktop quotes it in its diagnostics copy.
	PendingAuditEvents(ctx context.Context) (int, error)
	// OldestPendingAuditEvent reports when the oldest parked audit event entered
	// the outbox; ok is false on an empty outbox. A count alone may be normal on
	// a busy node, while an age that keeps growing means the drainer is stuck.
	OldestPendingAuditEvent(ctx context.Context) (time.Time, bool, error)
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

// PendingAuditEvents counts the outbox backlog. The event store creates its
// tables when the container builds it, so the table exists by the time a caller
// asks; a count failure is reported instead of silently reading zero.
func (c *Container) PendingAuditEvents(ctx context.Context) (int, error) {
	var pending int
	if err := c.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM pending_events`).Scan(&pending); err != nil {
		return 0, fmt.Errorf("count pending audit events: %w", err)
	}
	return pending, nil
}

// OldestPendingAuditEvent reads the oldest parked event's own timestamp, so the
// age a scraper derives is how long the event has waited in the outbox.
func (c *Container) OldestPendingAuditEvent(ctx context.Context) (time.Time, bool, error) {
	var stamp sql.NullInt64
	if err := c.DB.QueryRowContext(ctx, `SELECT MIN(timestamp) FROM pending_events`).Scan(&stamp); err != nil {
		return time.Time{}, false, fmt.Errorf("read oldest pending audit event: %w", err)
	}
	if !stamp.Valid {
		return time.Time{}, false, nil
	}
	return time.Unix(stamp.Int64, 0), true, nil
}

func (c *Container) TokenTransactions() voting.TransactionManager  { return c.TxManager }
func (c *Container) VotingTransactions() voting.TransactionManager { return c.VotingTx }
func (c *Container) NFTTransactions() nft.TransactionManager       { return c.NFTTx }
func (c *Container) Ledger() Chain                                 { return c.Chain }
func (c *Container) Fetcher() oracleapp.FetcherInterface           { return c.OracleFetcher }
