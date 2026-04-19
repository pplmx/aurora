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

type SyncEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	global   []Handler
}

func NewSyncEventBus() *SyncEventBus {
	return &SyncEventBus{
		handlers: make(map[string][]Handler),
	}
}

func (b *SyncEventBus) Publish(e events.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, h := range b.global {
		if err := h(e); err != nil {
			return err
		}
	}

	handlers := append([]Handler{}, b.handlers[e.EventType()]...)
	for _, h := range handlers {
		if err := h(e); err != nil {
			return err
		}
	}

	return nil
}

func (b *SyncEventBus) Subscribe(eventType string, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	idx := len(b.handlers[eventType]) - 1

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		handlers := b.handlers[eventType]
		if idx < len(handlers) {
			b.handlers[eventType] = append(handlers[:idx], handlers[idx+1:]...)
		}
	}
}

func (b *SyncEventBus) SubscribeAll(handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.global = append(b.global, handler)
	idx := len(b.global) - 1

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if idx < len(b.global) {
			b.global = append(b.global[:idx], b.global[idx+1:]...)
		}
	}
}
