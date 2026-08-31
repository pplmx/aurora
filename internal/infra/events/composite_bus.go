package events

import (
	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/logger"
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

	// The async legs are best-effort (ring buffer can be full, bus can be
	// closed); dropping the error silently hid that the event was lost from
	// those paths. Log it instead so a full/closed async bus is discoverable.
	if err := b.AsyncBus.Publish(e); err != nil {
		logger.Warn().Err(err).Str("event", e.EventType()).Msg("async event bus dropped event")
	}

	if err := b.PluginBus.Publish(e); err != nil {
		logger.Warn().Err(err).Str("event", e.EventType()).Msg("plugin event bus publish failed")
	}

	return nil
}

func (b *CompositeEventBus) Subscribe(eventType string, handler Handler) func() {
	return b.SyncBus.Subscribe(eventType, handler)
}

func (b *CompositeEventBus) SubscribeAll(handler Handler) func() {
	return b.SyncBus.SubscribeAll(handler)
}

var _ EventBus = (*CompositeEventBus)(nil)
