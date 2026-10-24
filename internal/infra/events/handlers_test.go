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
