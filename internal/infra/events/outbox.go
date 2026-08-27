package events

import (
	"context"
	"time"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/logger"
)

// PendingStore is the durable outbox surface the drainer drives. SQLiteEventStore
// implements it (the pending_events table, event_store.go); a fake satisfies it
// in tests without a real DB.
type PendingStore interface {
	SaveIdempotent(event events.Event) error
	ListDuePending(now time.Time, limit int) ([]PendingEvent, error)
	DropPending(id string) error
	BackoffPending(id string, attempts int, nextTry time.Time) error
}

const (
	// outboxBatch caps how many pending events one drain pass attempts. A large
	// backlog (e.g. after a long outage) drains in bounded waves instead of one
	// unbounded transaction-sized burst.
	outboxBatch = 50
	// outboxBackoffBase is the first gap after one failed retry.
	outboxBackoffBase = time.Second
	// outboxBackoffCap bounds the exponential backoff so a persistently-failing
	// destination does not hit events() every tick forever. Mirrors the oracle
	// scheduler's maxBackoff.
	outboxBackoffCap = 5 * time.Minute
)

// OutboxDrainer retries pending audit events whose direct delivery failed
// (TASK-119, ISS-111): v1.82 made a committed token op report (not) as failed
// when only the post-commit audit publish fails, but nothing healed the loss —
// SyncEventBus.Publish was fire-and-forget with the sole durability being the
// handler's autocommit INSERT at delivery time, so a transient SQLITE_BUSY /
// handler error permanently dropped the audit record. The handler now parks a
// failed delivery in pending_events; the drainer retries it with SaveIdempotent
// (UNIQUE-safe via the event UUID) on an exponential backoff until it lands.
//
// Run blocks until ctx is cancelled; DrainOnce does a single pass synchronously
// (used by short-lived CLI commands instead of a goroutine).
type OutboxDrainer struct {
	store PendingStore
	now   func() time.Time
}

// NewOutboxDrainer returns a drainer over store. now is injectable for
// deterministic tests; nil falls back to time.Now.
func NewOutboxDrainer(store PendingStore, now func() time.Time) *OutboxDrainer {
	if now == nil {
		now = time.Now
	}
	return &OutboxDrainer{store: store, now: now}
}

// Run drains pending events on a fixed cadence until ctx is cancelled. The
// cadence (1s) paces the retry loop; the per-event exponential backoff in
// BackoffPending is what actually stretches a stuck event's retry gap, so a
// fast tick is not a fast retry for a failing destination.
func (d *OutboxDrainer) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := d.DrainOnce(); err != nil {
				logger.Warn().Err(err).Msg("audit outbox drain pass failed")
			}
		}
	}
}

// DrainOnce retries every due pending event exactly once and returns the number
// that landed in events(). Events that still fail get an incremented attempt
// count and a backed-off next_attempt_at; they remain in the outbox. Batch
// reads are bounded (outboxBatch) so each pass is small and repeatable.
func (d *OutboxDrainer) DrainOnce() (int, error) {
	now := d.now()
	pending, err := d.store.ListDuePending(now, outboxBatch)
	if err != nil {
		return 0, err
	}

	drained := 0
	for _, p := range pending {
		if err := d.store.SaveIdempotent(p.Event); err != nil {
			attempts := p.Attempts + 1
			nextTry := now.Add(backoff(outboxBackoffBase, outboxBackoffCap, attempts))
			if berr := d.store.BackoffPending(p.Event.ID(), attempts, nextTry); berr != nil {
				logger.Warn().Err(berr).Str("event_id", p.Event.ID()).
					Msg("audit outbox: failed to backoff pending event")
			}
			continue
		}
		if err := d.store.DropPending(p.Event.ID()); err != nil {
			logger.Warn().Err(err).Str("event_id", p.Event.ID()).
				Msg("audit outbox: landed but failed to drop pending row")
		}
		drained++
	}
	return drained, nil
}

// backoff returns the retry gap for attempt n: base * 2^(n-1), capped at cap.
func backoff(base, cap time.Duration, attempts int) time.Duration {
	if attempts <= 0 {
		attempts = 1
	}
	gap := base
	for i := 1; i < attempts && gap < cap; i++ {
		gap *= 2
	}
	if gap > cap {
		return cap
	}
	return gap
}
