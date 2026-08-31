package events

import (
	"sync"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/logger"
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
	// Snapshot the handler list under the read lock, then run the handlers
	// outside it. Invoking handlers while holding RLock self-deadlocks the
	// whole bus when a handler calls Subscribe/Unsubscribe (each takes the
	// write lock) during publish — a latent trap in this public API (ISS-128).
	// The snapshot preserves per-publish ordering and the all-or-nothing
	// early-return contract: a handler registered mid-publish does not observe
	// the in-flight event (an RWMutex read lock already excluded concurrent
	// subscribers from the in-flight iteration), matching prior behavior.
	b.mu.RLock()
	globals := make([]Handler, 0, len(b.global))
	for _, s := range b.global {
		globals = append(globals, s.handler)
	}
	typed := make([]Handler, 0, len(b.handlers[e.EventType()]))
	for _, s := range b.handlers[e.EventType()] {
		typed = append(typed, s.handler)
	}
	b.mu.RUnlock()

	for _, h := range globals {
		if err := runHandler(h, e); err != nil {
			return err
		}
	}

	for _, h := range typed {
		if err := runHandler(h, e); err != nil {
			return err
		}
	}

	return nil
}

// runHandler invokes one subscriber safely: a panicking handler is recovered
// and logged so it can neither kill the async dispatcher goroutine (which
// would silently stop the whole bus forever) nor blow through a sync caller
// (the API server) after the operation already committed — the caller would
// otherwise surface a 500 the committed write did not deserve. A returned
// error still stops the chain (the all-or-nothing error contract is kept).
func runHandler(h Handler, e events.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error().Interface("panic", r).Str("event", e.EventType()).Msg("event bus subscriber panicked")
		}
	}()
	return h(e)
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
