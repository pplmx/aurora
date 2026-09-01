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
	// mu serializes Publish against Close. Publish's closed-check + channel
	// send, and Close's closed-store + done-close, both run under it, so a
	// publish can never succeed (return nil) after the consumer goroutine has
	// drained and exited: either it lands before done is closed (and is drained)
	// or it observes closed and returns ErrEventBusClosed. Without it,
	// Publish's check-then-act could slip a send in after processLoop had
	// already exited — silently dropping the event while reporting success
	// (ISS-243, TASK-245).
	mu sync.Mutex
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
	b.mu.Lock()
	defer b.mu.Unlock()

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
	// Hold the mutex so no in-flight Publish is past its closed-check while we
	// flip the flag and stop the consumer: a send cannot be accepted after
	// done is closed and processLoop has been asked to exit (ISS-243, TASK-245).
	b.mu.Lock()
	if b.closed.Swap(true) {
		b.mu.Unlock()
		return
	}
	close(b.done)
	b.mu.Unlock()

	// Drain what's already queued (the done-drain loop in processLoop).
	b.wg.Wait()
}
