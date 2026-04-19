package events

import (
	"errors"
	"sync"
	"testing"
	"time"

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

func TestCompositeEventBus_Publish(t *testing.T) {
	bus := NewCompositeEventBus()

	var syncCalled, asyncCalled, pluginCalled bool
	var mu sync.Mutex

	bus.SyncBus.Subscribe("test.event", func(e events.Event) error {
		mu.Lock()
		syncCalled = true
		mu.Unlock()
		return nil
	})

	bus.AsyncBus.Subscribe("test.event", func(e events.Event) error {
		mu.Lock()
		asyncCalled = true
		mu.Unlock()
		return nil
	})

	bus.PluginBus.Subscribe("test.event", func(e events.Event) error {
		mu.Lock()
		pluginCalled = true
		mu.Unlock()
		return nil
	})

	e := newMockEvent("test.event", "agg-1")
	if err := bus.Publish(e); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if !syncCalled {
		t.Error("sync bus handler was not called")
	}
	if !asyncCalled {
		t.Error("async bus handler was not called")
	}
	if !pluginCalled {
		t.Error("plugin bus handler was not called")
	}
	mu.Unlock()
}

func TestCompositeEventBus_SyncErrorStopsAsyncPlugin(t *testing.T) {
	bus := NewCompositeEventBus()

	var asyncCalled, pluginCalled bool
	bus.AsyncBus.Subscribe("error.event", func(e events.Event) error {
		asyncCalled = true
		return nil
	})

	bus.PluginBus.Subscribe("error.event", func(e events.Event) error {
		pluginCalled = true
		return nil
	})

	bus.SyncBus.Subscribe("error.event", func(e events.Event) error {
		return errors.New("sync error")
	})

	e := newMockEvent("error.event", "agg-1")
	err := bus.Publish(e)
	if err == nil {
		t.Fatal("expected error from sync bus")
	}
	if err.Error() != "sync error" {
		t.Errorf("expected 'sync error', got '%s'", err.Error())
	}

	if asyncCalled {
		t.Error("async bus handler should not be called when sync bus errors")
	}
	if pluginCalled {
		t.Error("plugin bus handler should not be called when sync bus errors")
	}
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

	handler.Handle(e1)
	handler.Handle(e2)
	handler.Handle(e3)

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

	handler.Handle(newMockEvent("a", "agg-1"))
	handler.Handle(newMockEvent("a", "agg-2"))
	handler.Handle(newMockEvent("b", "agg-3"))

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
