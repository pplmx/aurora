package events

import (
	"sync"
	"sync/atomic"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/logger"
)

type AsyncEventBus struct {
	bus       *SyncEventBus
	eventChan chan events.Event
	done      chan struct{}
	wg        sync.WaitGroup
	closed    atomic.Bool
}

func NewAsyncEventBus(bufSize int) *AsyncEventBus {
	bus := &AsyncEventBus{
		bus:       NewSyncEventBus(),
		eventChan: make(chan events.Event, bufSize),
		done:      make(chan struct{}),
	}

	bus.wg.Add(1)
	go bus.processLoop()

	return bus
}

func (b *AsyncEventBus) processLoop() {
	defer b.wg.Done()
	for {
		select {
		case e := <-b.eventChan:
			if err := b.bus.Publish(e); err != nil {
				logger.Warn().Err(err).Str("event", e.EventType()).Msg("async event bus publish failed")
			}
		case <-b.done:
			for {
				select {
				case e := <-b.eventChan:
					if err := b.bus.Publish(e); err != nil {
						logger.Warn().Err(err).Str("event", e.EventType()).Msg("async event bus drain publish failed")
					}
				default:
					return
				}
			}
		}
	}
}

func (b *AsyncEventBus) Publish(e events.Event) error {
	if b.closed.Load() {
		return events.ErrEventBusClosed
	}

	select {
	case b.eventChan <- e:
		return nil
	default:
		return events.ErrEventBusFull
	}
}

func (b *AsyncEventBus) Subscribe(eventType string, handler Handler) func() {
	return b.bus.Subscribe(eventType, handler)
}

func (b *AsyncEventBus) SubscribeAll(handler Handler) func() {
	return b.bus.SubscribeAll(handler)
}

func (b *AsyncEventBus) Close() {
	if b.closed.Swap(true) {
		return
	}
	close(b.done)
	b.wg.Wait()
}
