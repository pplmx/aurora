package events

import (
	"sync"

	"github.com/pplmx/aurora/internal/domain/events"
)

type AuditStore interface {
	Save(event events.Event) error
}

type AuditHandler struct {
	store AuditStore
}

func NewAuditHandler(store AuditStore) *AuditHandler {
	return &AuditHandler{store: store}
}

func (h *AuditHandler) Handle(e events.Event) error {
	return h.store.Save(e)
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
