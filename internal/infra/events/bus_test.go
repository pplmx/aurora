package events

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/stretchr/testify/require"
)

type mockEvent struct {
	*events.BaseEvent
}

func newMockEvent(eventType, aggID string) *mockEvent {
	return &mockEvent{
		BaseEvent: events.NewBaseEvent(eventType, aggID, []byte(`{}`)),
	}
}

func TestSyncEventBus_Publish_CallsHandlers(t *testing.T) {
	bus := NewSyncEventBus()

	called := false
	handler := func(e events.Event) error {
		called = true
		return nil
	}

	bus.Subscribe("test.event", handler)

	err := bus.Publish(newMockEvent("test.event", "agg-1"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestSyncEventBus_Publish_AllOrNothing(t *testing.T) {
	bus := NewSyncEventBus()

	callOrder := []int{}
	handler1 := func(e events.Event) error {
		callOrder = append(callOrder, 1)
		return nil
	}
	handler2 := func(e events.Event) error {
		callOrder = append(callOrder, 2)
		return errors.New("handler 2 failed")
	}
	handler3 := func(e events.Event) error {
		callOrder = append(callOrder, 3)
		return nil
	}

	bus.Subscribe("test.event", handler1)
	bus.Subscribe("test.event", handler2)
	bus.Subscribe("test.event", handler3)

	err := bus.Publish(newMockEvent("test.event", "agg-1"))
	if err == nil {
		t.Fatal("expected error from handler2")
	}

	if len(callOrder) != 2 {
		t.Fatalf("expected 2 handlers called (1 and 2), got %d: %v", len(callOrder), callOrder)
	}
	if callOrder[0] != 1 || callOrder[1] != 2 {
		t.Fatalf("expected handlers 1 then 2 to be called, got %v", callOrder)
	}
}

func TestSyncEventBus_Publish_GlobalHandlersFirst(t *testing.T) {
	bus := NewSyncEventBus()

	callOrder := []string{}
	globalHandler := func(e events.Event) error {
		callOrder = append(callOrder, "global")
		return nil
	}
	typeHandler := func(e events.Event) error {
		callOrder = append(callOrder, "type")
		return nil
	}

	bus.SubscribeAll(globalHandler)
	bus.Subscribe("test.event", typeHandler)

	err := bus.Publish(newMockEvent("test.event", "agg-1"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(callOrder) != 2 {
		t.Fatalf("expected 2 handlers called, got %d: %v", len(callOrder), callOrder)
	}
	if callOrder[0] != "global" || callOrder[1] != "type" {
		t.Fatalf("expected global then type, got %v", callOrder)
	}
}

func TestSyncEventBus_Subscribe_ReturnsUnsubscribe(t *testing.T) {
	bus := NewSyncEventBus()

	called := false
	handler := func(e events.Event) error {
		called = true
		return nil
	}

	unsubscribe := bus.Subscribe("test.event", handler)
	_ = bus.Publish(newMockEvent("test.event", "agg-1"))

	if !called {
		t.Fatal("expected handler to be called before unsubscribe")
	}

	called = false
	unsubscribe()
	_ = bus.Publish(newMockEvent("test.event", "agg-1"))

	if called {
		t.Fatal("expected handler NOT to be called after unsubscribe")
	}
}

func TestSyncEventBus_Subscribe_MultipleHandlers(t *testing.T) {
	bus := NewSyncEventBus()

	results := []int{}
	bus.Subscribe("test.event", func(e events.Event) error {
		results = append(results, 1)
		return nil
	})
	bus.Subscribe("test.event", func(e events.Event) error {
		results = append(results, 2)
		return nil
	})
	bus.Subscribe("test.event", func(e events.Event) error {
		results = append(results, 3)
		return nil
	})

	err := bus.Publish(newMockEvent("test.event", "agg-1"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
}

func TestSyncEventBus_SubscribeAll_SubscribesToAllEvents(t *testing.T) {
	bus := NewSyncEventBus()

	calls := 0
	handler := func(e events.Event) error {
		calls++
		return nil
	}

	bus.SubscribeAll(handler)

	_ = bus.Publish(newMockEvent("event.one", "agg-1"))
	_ = bus.Publish(newMockEvent("event.two", "agg-2"))
	_ = bus.Publish(newMockEvent("event.three", "agg-3"))

	if calls != 3 {
		t.Fatalf("expected handler called 3 times, got %d", calls)
	}
}

func TestSyncEventBus_SubscribeAll_ReturnsUnsubscribe(t *testing.T) {
	bus := NewSyncEventBus()

	calls := 0
	handler := func(e events.Event) error {
		calls++
		return nil
	}

	unsubscribe := bus.SubscribeAll(handler)

	_ = bus.Publish(newMockEvent("event.one", "agg-1"))
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}

	unsubscribe()
	_ = bus.Publish(newMockEvent("event.two", "agg-2"))
	if calls != 1 {
		t.Fatalf("expected still 1 call after unsubscribe, got %d", calls)
	}
}

func TestSyncEventBus_Unsubscribe_TypeSpecific(t *testing.T) {
	bus := NewSyncEventBus()

	calls := 0
	handler := func(e events.Event) error {
		calls++
		return nil
	}

	unsubscribe := bus.Subscribe("specific.event", handler)
	bus.SubscribeAll(handler)

	_ = bus.Publish(newMockEvent("specific.event", "agg-1"))
	if calls != 2 {
		t.Fatalf("expected 2 calls (type + global), got %d", calls)
	}

	unsubscribe()
	calls = 0
	_ = bus.Publish(newMockEvent("specific.event", "agg-1"))
	if calls != 1 {
		t.Fatalf("expected 1 call (global only), got %d", calls)
	}
}

func TestSyncEventBus_Unsubscribe_MultipleOfSameType(t *testing.T) {
	bus := NewSyncEventBus()

	calls := 0
	handler1 := func(e events.Event) error {
		calls++
		return nil
	}
	handler2 := func(e events.Event) error {
		calls += 10
		return nil
	}

	unsubscribe1 := bus.Subscribe("test.event", handler1)
	bus.Subscribe("test.event", handler2)

	_ = bus.Publish(newMockEvent("test.event", "agg-1"))
	if calls != 11 {
		t.Fatalf("expected 11, got %d", calls)
	}

	unsubscribe1()
	calls = 0
	_ = bus.Publish(newMockEvent("test.event", "agg-1"))
	if calls != 10 {
		t.Fatalf("expected 10, got %d", calls)
	}
}

func TestSyncEventBus_Publish_NoSubscribers(t *testing.T) {
	bus := NewSyncEventBus()

	err := bus.Publish(newMockEvent("test.event", "agg-1"))
	if err != nil {
		t.Fatalf("expected no error with no subscribers, got %v", err)
	}
}

func TestSyncEventBus_Publish_TypeMismatch(t *testing.T) {
	bus := NewSyncEventBus()

	called := false
	bus.Subscribe("correct.event", func(e events.Event) error {
		called = true
		return nil
	})

	_ = bus.Publish(newMockEvent("wrong.event", "agg-1"))
	if called {
		t.Fatal("handler for different event type should not be called")
	}
}

func TestSyncEventBus_GlobalHandlerError_StopsTypeHandlers(t *testing.T) {
	bus := NewSyncEventBus()

	callOrder := []string{}
	bus.SubscribeAll(func(e events.Event) error {
		callOrder = append(callOrder, "global")
		return errors.New("global error")
	})
	bus.Subscribe("test.event", func(e events.Event) error {
		callOrder = append(callOrder, "type")
		return nil
	})

	err := bus.Publish(newMockEvent("test.event", "agg-1"))
	if err == nil {
		t.Fatal("expected error from global handler")
	}

	if len(callOrder) != 1 {
		t.Fatalf("expected only global handler to be called, got %v", callOrder)
	}
	if callOrder[0] != "global" {
		t.Fatalf("expected global handler first, got %v", callOrder)
	}
}

func TestNewSyncEventBus(t *testing.T) {
	bus := NewSyncEventBus()
	require.NotNil(t, bus)
	require.NotNil(t, bus.handlers)
	require.Equal(t, 0, len(bus.handlers))
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewSyncEventBus()

	var count int32
	var mu sync.Mutex
	seen := make(map[string]bool)

	bus.SubscribeAll(func(e events.Event) error {
		mu.Lock()
		seen[e.ID()] = true
		mu.Unlock()
		atomic.AddInt32(&count, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bus.Publish(events.NewBaseEvent("test", "agg", nil))
		}()
	}
	wg.Wait()

	require.Equal(t, int32(50), atomic.LoadInt32(&count))
	mu.Lock()
	require.Len(t, seen, 50)
	mu.Unlock()
}

func TestEventBus_ConcurrentSubscribe(t *testing.T) {
	bus := NewSyncEventBus()
	var count int32

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe("test", func(e events.Event) error {
				atomic.AddInt32(&count, 1)
				return nil
			})
		}()
	}
	wg.Wait()

	_ = bus.Publish(events.NewBaseEvent("test", "agg", nil))
	require.GreaterOrEqual(t, atomic.LoadInt32(&count), int32(10))
}

func TestEventBus_HandlerChainAllOrNothing(t *testing.T) {
	bus := NewSyncEventBus()
	var order []string

	bus.SubscribeAll(func(e events.Event) error {
		order = append(order, "handler1")
		return nil
	})
	bus.SubscribeAll(func(e events.Event) error {
		order = append(order, "handler2")
		return errors.New("handler2 failed")
	})
	bus.SubscribeAll(func(e events.Event) error {
		order = append(order, "handler3")
		return nil
	})

	err := bus.Publish(events.NewBaseEvent("test", "agg", nil))
	require.Error(t, err)
	require.Contains(t, err.Error(), "handler2 failed")
	require.Equal(t, []string{"handler1", "handler2"}, order)
	require.Len(t, order, 2)
}
