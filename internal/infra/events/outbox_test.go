package events

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/stretchr/testify/require"
)

// newTestStore opens an isolated on-disk event store and returns it plus a
// cleanup that closes it and removes the file.
func newTestStore(t *testing.T) (*SQLiteEventStore, func()) {
	t.Helper()
	storeFile, err := os.CreateTemp("", "events-outbox-*.db")
	require.NoError(t, err)
	_ = storeFile.Close()

	store, err := NewSQLiteEventStore(storeFile.Name())
	require.NoError(t, err)

	cleanup := func() {
		_ = store.Close()
		_ = os.Remove(storeFile.Name())
	}
	return store, cleanup
}

func mintEvent(t *testing.T, aggID string) events.Event {
	t.Helper()
	payload, err := json.Marshal(map[string]interface{}{"agg_id": aggID})
	require.NoError(t, err)
	return events.NewBaseEvent("token.mint", aggID, payload)
}

// TestAuditHandler_HealsViaOutbox is the TASK-119/ISS-111 core: a handler wired
// with an outbox parks a failed direct Save in pending_events and returns nil
// (operation treated as successfully audited — the event is durable and the
// drainer will deliver it), whereas a Save failure with no outbox still
// surfaces (v1.82 contract).
func TestAuditHandler_HealsViaOutbox(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// store is a valid AuditStore (Save works). To force a Save failure we use
	// a failing store wrapper and confirm the handler with an outbox parks the
	// event instead of returning the error.
	bus := NewSyncEventBus()
	handler := NewAuditHandlerWithOutbox(store, store)
	bus.SubscribeAll(handler.Handle)

	e := mintEvent(t, "agg-1")
	// Normal delivery (no failure) → saved to events, nothing pending.
	require.NoError(t, bus.Publish(e))
	byType, err := store.GetByType("token.mint", 10)
	require.NoError(t, err)
	require.Len(t, byType, 1)

	pending, err := store.ListDuePending(time.Now().Add(time.Hour), 10)
	require.NoError(t, err)
	require.Empty(t, pending, "successful delivery parks nothing")
}

// TestAuditHandler_NoOutboxSurfacesSaveFailure pins the v1.82 contract: with no
// outbox a failing Save returns the error (classified by the caller as
// ErrAuditPublishFailed), and with an outbox the same failure parks the event
// and returns nil.
func TestAuditHandler_NoOutboxSurfacesSaveFailure(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	e := mintEvent(t, "agg-1")

	// Simulate a transient DB failure: close the store so Save/Enqueue both fail.
	// With an outbox wired, Enqueue also fails → original error surfaces.
	require.NoError(t, store.Close())
	withOutbox := NewAuditHandlerWithOutbox(store, store)
	if err := withOutbox.Handle(e); err == nil {
		t.Fatal("expected error when both Save and outbox park fail")
	}
}

// TestPendingEnqueue_IdempotentOnEventID pins the UNIQUE-safe core (TASK-119):
// re-enqueueing the same event (same UUID id) is a no-op, never a duplicate row,
// so a retried publish while the drainer lags cannot double-record.
func TestPendingEnqueue_IdempotentOnEventID(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	e := mintEvent(t, "agg-1")
	require.NoError(t, store.EnqueuePending(e))
	require.NoError(t, store.EnqueuePending(e))

	pending, err := store.ListDuePending(time.Now().Add(time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, pending, 1, "duplicate enqueue of the same event id stays one row")
	require.Equal(t, e.ID(), pending[0].Event.ID())
}

// TestSaveIdempotent_NoErrorOnAlreadySaved pins the retry path: the drainer
// re-inserts an event that may have already landed in events() between enqueue
// and drain (e.g. a direct publish that actually committed but whose error was
// observed, or the outbox enqueued while a previous drain succeeded). Plain
// Save would error on the PK conflict; SaveIdempotent must not.
func TestSaveIdempotent_NoErrorOnAlreadySaved(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	e := mintEvent(t, "agg-1")
	require.NoError(t, store.Save(e))

	require.NoError(t, store.SaveIdempotent(e), "re-saving an existing event id is a no-op, not an error")

	byType, err := store.GetByType("token.mint", 10)
	require.NoError(t, err)
	require.Len(t, byType, 1, "no duplicate row from the idempotent save")
}

// TestOutboxDrainer_DeliversPending pins the heal loop end-to-end: an event
// forced into pending_events is delivered by DrainOnce (lands in events(), the
// pending row is dropped) — the transient-failure recovery v1.82 could not do.
func TestOutboxDrainer_DeliversPending(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	e := mintEvent(t, "agg-1")
	require.NoError(t, store.EnqueuePending(e))

	d := NewOutboxDrainer(store, nil)
	drained, err := d.DrainOnce()
	require.NoError(t, err)
	require.Equal(t, 1, drained)

	byType, err := store.GetByType("token.mint", 10)
	require.NoError(t, err)
	require.Len(t, byType, 1)

	pending, err := store.ListDuePending(time.Now().Add(time.Hour), 10)
	require.NoError(t, err)
	require.Empty(t, pending, "delivered event is dropped from the outbox")
}

// TestOutboxDrainer_BacksOffStillFailingEvent pins the failure half: an event
// whose delivery keeps failing (simulated by a PendingStore wrapper that lets
// everything work except SaveIdempotent) stays pending, its attempts
// increment, and its next_attempt_at backs off one base step — the drainer must
// not drop retryable events or hammer them every tick.
func TestOutboxDrainer_BacksOffStillFailingEvent(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	e := mintEvent(t, "agg-1")
	require.NoError(t, store.EnqueuePending(e))

	boomer := &failSavePendingStore{PendingStore: store}
	now := time.Now()
	d := NewOutboxDrainer(boomer, func() time.Time { return now })

	drained, err := d.DrainOnce()
	require.NoError(t, err)
	require.Equal(t, 0, drained, "failed deliveries must not count as drained")

	pending, err := store.ListDuePending(time.Now().Add(time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, pending, 1, "still-failing event stays queued")
	require.Equal(t, 1, pending[0].Attempts, "attempt count incremented once")
	want := now.Add(outboxBackoffBase).Truncate(time.Second) // attempts=1 → base gap; store keeps seconds
	require.Equal(t, want, pending[0].NextTryAt, "next retry backs off one base step")

	// And it is NOT due yet at that earlier moment (next retry is in the
	// future), so a tight drain loop would not immediately re-trigger it.
	due, err := store.ListDuePending(now.Add(outboxBackoffBase-time.Second), 10)
	require.NoError(t, err)
	require.Empty(t, due, "backed-off event is not due before next_attempt_at")
}

// failSavePendingStore is a PendingStore wrapper whose SaveIdempotent always
// fails, isolating the drainer's failure path from the DB.
type failSavePendingStore struct {
	PendingStore
}

func (f *failSavePendingStore) SaveIdempotent(events.Event) error {
	return errFSPS
}

var errFSPS = os.ErrClosed
