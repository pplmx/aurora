package events

import (
	"sync"

	"github.com/pplmx/aurora/internal/domain/events"
)

type AuditStore interface {
	Save(event events.Event) error
}

// OutboxWriter is where AuditHandler parks a failed delivery so a background
// OutboxDrainer can retry it (TASK-119, ISS-111). The pending_events table via
// SQLiteEventStore.EnqueuePending implements it; the contract is "durably
// record this event for later retry", idempotent on the event id.
type OutboxWriter interface {
	EnqueuePending(event events.Event) error
}

type AuditHandler struct {
	store  AuditStore
	outbox OutboxWriter // nil when the call site deliberately disables healing
}

// NewAuditHandler returns a handler that persists audit events straight to
// store. A failed Save surfaces immediately — the v1.82 contract (a committed
// op whose audit publish fails is reported as that distinct failure). Callers
// that also wire an outbox should use NewAuditHandlerWithOutbox.
func NewAuditHandler(store AuditStore) *AuditHandler {
	return &AuditHandler{store: store}
}

// NewAuditHandlerWithOutbox heals transient delivery failures instead of just
// reporting them: on a Save error the event is parked in the durable outbox
// (EnqueuePending) and the handler returns nil — the audit trail is durably
// recorded and the OutboxDrainer will deliver it. Only if BOTH the direct Save
// AND the outbox park fail (e.g. the DB itself is down) does the handler return
// an error, preserving the ErrAuditPublishFailed classification. The v1.82
// do-not-retry semantics still hold: returning nil here means "already
// durably captured", never "retry the operation".
func NewAuditHandlerWithOutbox(store AuditStore, outbox OutboxWriter) *AuditHandler {
	return &AuditHandler{store: store, outbox: outbox}
}

func (h *AuditHandler) Handle(e events.Event) error {
	if err := h.store.Save(e); err != nil {
		if h.outbox == nil {
			return err
		}
		// Park for the drainer. If even this fails the audit record is
		// genuinely lost — surface the original Save error so the caller
		// classifies it as ErrAuditPublishFailed (never a retryable op error).
		if perr := h.outbox.EnqueuePending(e); perr != nil {
			return err
		}
		return nil
	}
	return nil
}

type StatsHandler struct {
	mu     sync.RWMutex
	counts map[string]int64
}

func NewStatsHandler() *StatsHandler {
	return &StatsHandler{
		counts: make(map[string]int64),
	}
}

func (h *StatsHandler) Handle(e events.Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.counts[e.EventType()]++
	return nil
}

func (h *StatsHandler) GetCount(eventType string) int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.counts[eventType]
}

func (h *StatsHandler) GetAllCounts() map[string]int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	counts := make(map[string]int64, len(h.counts))
	for k, v := range h.counts {
		counts[k] = v
	}
	return counts
}
