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
	// PendingAuditList reads the parked events themselves, so the window can show
	// what is stuck rather than only that something is.
	PendingAuditList(ctx context.Context, limit int) ([]PendingAuditEvent, error)
	// PendingAuditModules names the modules whose events are parked. The desktop
	// narrows the ledger to one of them when the operator follows the backlog,
	// so the page opens on the records that are stuck.
	PendingAuditModules(ctx context.Context) ([]string, error)
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

// PendingAuditEvent is one row of the outbox as the window reads it: enough to
// say what is stuck and how hard the drainer has been trying, without exposing
// the payload (which the ledger page already shows for the same operation).
type PendingAuditEvent struct {
	EventType string
	Module    string
	Aggregate string
	// ParkedAt is when the event entered the outbox, so the window can say how
	// long it has waited rather than only how many are waiting.
	ParkedAt time.Time
	// Attempts counts failed redeliveries; NextTryAt is when the drainer may
	// try again. Both come from the backoff the drainer keeps, so a row that
	// stops moving is visible as a growing attempts count.
	Attempts  int
	NextTryAt time.Time
	// Payload is the exact recorded bytes the drainer will read. The window shows
	// it on request rather than by default: it is the one fact that says which
	// record is parked, but it is JSON, and a wall of it in every row would bury
	// the operation name and the retry state that the row is scanned for.
	Payload string
}

// PendingAuditList reads the outbox newest-first, capped at limit. The window
// shows these rows so a backlog is inspectable rather than only countable: the
// footer says how many are parked, and this says which.
func (c *Container) PendingAuditList(ctx context.Context, limit int) ([]PendingAuditEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := c.DB.QueryContext(ctx, `
		SELECT event_type, module, agg_id, timestamp, attempts, next_attempt_at, payload
		FROM pending_events
		ORDER BY timestamp DESC, id ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("read pending audit events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []PendingAuditEvent
	for rows.Next() {
		var eventType, module, aggregate string
		var timestamp int64
		var attempts, nextTryAt int64
		var payload []byte
		if err := rows.Scan(&eventType, &module, &aggregate, &timestamp, &attempts, &nextTryAt, &payload); err != nil {
			return nil, fmt.Errorf("scan pending audit event: %w", err)
		}
		out = append(out, PendingAuditEvent{
			EventType: eventType,
			Module:    module,
			Aggregate: aggregate,
			ParkedAt:  time.Unix(timestamp, 0),
			Attempts:  int(attempts),
			NextTryAt: time.Unix(nextTryAt, 0),
			Payload:   string(payload),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read pending audit events: %w", err)
	}
	return out, nil
}

// PendingAuditModules lists the distinct modules whose parked events are waiting
// in the outbox. The window hands one of them to the ledger's search when the
// operator follows the backlog, so the page opens on the records that are stuck
// rather than on the whole chain — the backlog is usually one sink refusing one
// module's writes.
func (c *Container) PendingAuditModules(ctx context.Context) ([]string, error) {
	rows, err := c.DB.QueryContext(ctx,
		`SELECT DISTINCT module FROM pending_events WHERE module <> '' ORDER BY module`)
	if err != nil {
		return nil, fmt.Errorf("read pending audit modules: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var modules []string
	for rows.Next() {
		var module string
		if err := rows.Scan(&module); err != nil {
			return nil, fmt.Errorf("scan pending audit module: %w", err)
		}
		modules = append(modules, module)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read pending audit modules: %w", err)
	}
	return modules, nil
}

func (c *Container) TokenTransactions() voting.TransactionManager  { return c.TxManager }
func (c *Container) VotingTransactions() voting.TransactionManager { return c.VotingTx }
func (c *Container) NFTTransactions() nft.TransactionManager       { return c.NFTTx }
func (c *Container) Ledger() Chain                                 { return c.Chain }
func (c *Container) Fetcher() oracleapp.FetcherInterface           { return c.OracleFetcher }
