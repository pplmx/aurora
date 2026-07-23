package events

import (
	"sync"

	"github.com/pplmx/aurora/internal/domain/events"
)

type EventBus interface {
	Publish(events.Event) error
	Subscribe(eventType string, handler Handler) func()
	SubscribeAll(handler Handler) func()
}

type Handler func(events.Event) error

// subscription carries enough information to find and remove the
// handler regardless of how the underlying slice has shifted since
// it was registered. Comparing by handler identity (rather than by
// index) closes the "stale index" bug: if Subscribe(S1, A) →
// Unsubscribe(A) → Subscribe(S1, B), B would land at the index
// A's unsubscribe closure captured, and removing that index would
// silently evict B.
//
// Keeping the per-type list as a slice of *subscription (pointer
// values) gives both stable identity and O(1) lookup once the
// handler is found.
type subscription struct {
	eventType string // empty for global subscriptions
	handler   Handler
}

type SyncEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]*subscription
	global   []*subscription
}

func NewSyncEventBus() *SyncEventBus {
	return &SyncEventBus{
		handlers: make(map[string][]*subscription),
	}
}

// removeSubscription finds and deletes a single subscription by
// identity. It is O(n) per remove, which is fine for the
// notification-fanout sizes this bus handles (typically dozens,
// occasionally thousands). Returns true if a subscription was
// removed.
func (b *SyncEventBus) removeSubscription(sub *subscription) bool {
	if sub.eventType == "" {
		for i, s := range b.global {
			if s == sub {
				b.global = append(b.global[:i], b.global[i+1:]...)
				return true
			}
		}
		return false
	}
	subs := b.handlers[sub.eventType]
	for i, s := range subs {
		if s == sub {
			b.handlers[sub.eventType] = append(subs[:i], subs[i+1:]...)
			return true
		}
	}
	return false
}

func (b *SyncEventBus) Publish(e events.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, s := range b.global {
		if err := s.handler(e); err != nil {
			return err
		}
	}

	for _, s := range b.handlers[e.EventType()] {
		if err := s.handler(e); err != nil {
			return err
		}
	}

	return nil
}

func (b *SyncEventBus) Subscribe(eventType string, handler Handler) func() {
	sub := &subscription{eventType: eventType, handler: handler}

	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], sub)
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.removeSubscription(sub)
		})
	}
}

func (b *SyncEventBus) SubscribeAll(handler Handler) func() {
	sub := &subscription{handler: handler}

	b.mu.Lock()
	b.global = append(b.global, sub)
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.removeSubscription(sub)
		})
	}
}
