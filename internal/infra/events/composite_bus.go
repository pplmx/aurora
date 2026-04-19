package events

import (
	"github.com/pplmx/aurora/internal/domain/events"
)

type CompositeEventBus struct {
	SyncBus   *SyncEventBus
	AsyncBus  *AsyncEventBus
	PluginBus *SyncEventBus
}

func NewCompositeEventBus() *CompositeEventBus {
	return &CompositeEventBus{
		SyncBus:   NewSyncEventBus(),
		AsyncBus:  NewAsyncEventBus(100),
		PluginBus: NewSyncEventBus(),
	}
}

func (b *CompositeEventBus) Publish(e events.Event) error {
	if err := b.SyncBus.Publish(e); err != nil {
		return err
	}

	_ = b.AsyncBus.Publish(e)

	_ = b.PluginBus.Publish(e)

	return nil
}

func (b *CompositeEventBus) Subscribe(eventType string, handler Handler) func() {
	return b.SyncBus.Subscribe(eventType, handler)
}

func (b *CompositeEventBus) SubscribeAll(handler Handler) func() {
	return b.SyncBus.SubscribeAll(handler)
}

var _ EventBus = (*CompositeEventBus)(nil)
