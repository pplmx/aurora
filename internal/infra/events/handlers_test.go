package events

import (
	"sync"
	"testing"

	"github.com/pplmx/aurora/internal/domain/events"
)

type mockAuditStore struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *mockAuditStore) Save(e events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *mockAuditStore) GetEvents() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]events.Event, len(s.events))
	copy(result, s.events)
	return result
}

func (s *mockAuditStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
}

func TestAuditHandler_Handle(t *testing.T) {
	store := &mockAuditStore{}
	handler := &AuditHandler{store: store}

	e := newMockEvent("audit.test", "agg-1")
	if err := handler.Handle(e); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	stored := store.GetEvents()
	if len(stored) != 1 {
		t.Fatalf("expected 1 event, got %d", len(stored))
	}
	if stored[0].ID() != e.ID() {
		t.Errorf("expected event ID %s, got %s", e.ID(), stored[0].ID())
	}
}

func TestStatsHandler_Handle(t *testing.T) {
	handler := NewStatsHandler()

	e1 := newMockEvent("stats.click", "agg-1")
	e2 := newMockEvent("stats.click", "agg-2")
	e3 := newMockEvent("stats.view", "agg-3")

	_ = handler.Handle(e1)
	_ = handler.Handle(e2)
	_ = handler.Handle(e3)

	count := handler.GetCount("stats.click")
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	count = handler.GetCount("stats.view")
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	count = handler.GetCount("stats.missing")
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestStatsHandler_GetAllCounts(t *testing.T) {
	handler := NewStatsHandler()

	_ = handler.Handle(newMockEvent("a", "agg-1"))
	_ = handler.Handle(newMockEvent("a", "agg-2"))
	_ = handler.Handle(newMockEvent("b", "agg-3"))

	all := handler.GetAllCounts()
	if len(all) != 2 {
		t.Errorf("expected 2 event types, got %d", len(all))
	}
	if all["a"] != 2 {
		t.Errorf("expected a=2, got %d", all["a"])
	}
	if all["b"] != 1 {
		t.Errorf("expected b=1, got %d", all["b"])
	}
}
