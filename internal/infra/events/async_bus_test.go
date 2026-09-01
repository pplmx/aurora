package events

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsyncEventBus_PublishNonBlocking(t *testing.T) {
	bus := NewAsyncEventBus(10)

	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe("test.event", func(e events.Event) error {
		wg.Done()
		return nil
	})

	start := time.Now()
	err := bus.Publish(events.NewBaseEvent("test.event", "agg1", []byte(`{}`)))
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, 50*time.Millisecond, "Publish should be non-blocking")

	wg.Wait()
	bus.Close()
}

func TestAsyncEventBus_ClosePreventsPublish(t *testing.T) {
	bus := NewAsyncEventBus(10)
	bus.Close()

	err := bus.Publish(events.NewBaseEvent("test.event", "agg1", []byte(`{}`)))
	assert.ErrorIs(t, err, events.ErrEventBusClosed)
}

func TestAsyncEventBus_CloseDrainsRemainingEvents(t *testing.T) {
	bus := NewAsyncEventBus(10)
	defer bus.Close()

	var received int
	var mu sync.Mutex

	bus.Subscribe("test.event", func(e events.Event) error {
		mu.Lock()
		received++
		mu.Unlock()
		return nil
	})

	for i := 0; i < 5; i++ {
		_ = bus.Publish(events.NewBaseEvent("test.event", "agg1", []byte(`{}`)))
	}

	bus.Close()

	mu.Lock()
	assert.Equal(t, 5, received)
	mu.Unlock()
}

func TestAsyncEventBus_PanickingSubscriberSurvives(t *testing.T) {
	bus := NewAsyncEventBus(10)

	var healthy sync.WaitGroup
	healthy.Add(2)
	bus.SubscribeAll(func(e events.Event) error {
		panic("boom")
	})
	bus.Subscribe("test.event", func(e events.Event) error {
		healthy.Done()
		return nil
	})

	for i := 0; i < 2; i++ {
		require.NoError(t, bus.Publish(events.NewBaseEvent("test.event", "agg-1", []byte(`{}`))))
	}

	// The panicking subscriber must not kill the dispatcher goroutine: the
	// healthy one still receives every event. Before the recover it silently
	// stopped processing forever after the first panic.
	done := make(chan struct{})
	go func() { healthy.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatcher stopped after a panicking subscriber; healthy subscriber missed events")
	}
	bus.Close()
}

func TestAsyncEventBus_Subscribe(t *testing.T) {
	bus := NewAsyncEventBus(10)
	defer bus.Close()

	var received int64
	unsubscribe := bus.Subscribe("test.event", func(e events.Event) error {
		atomic.AddInt64(&received, 1)
		return nil
	})

	_ = bus.Publish(events.NewBaseEvent("test.event", "agg1", []byte(`{}`)))
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int64(1), atomic.LoadInt64(&received))

	unsubscribe()

	_ = bus.Publish(events.NewBaseEvent("test.event", "agg1", []byte(`{}`)))
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int64(1), atomic.LoadInt64(&received), "Event should not be received after unsubscribe")
}

func TestAsyncEventBus_SubscribeAll(t *testing.T) {
	bus := NewAsyncEventBus(10)
	defer bus.Close()

	var received int64
	unsubscribe := bus.SubscribeAll(func(e events.Event) error {
		atomic.AddInt64(&received, 1)
		return nil
	})

	_ = bus.Publish(events.NewBaseEvent("test.event", "agg1", []byte(`{}`)))
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int64(1), atomic.LoadInt64(&received))

	unsubscribe()

	_ = bus.Publish(events.NewBaseEvent("test.event", "agg1", []byte(`{}`)))
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int64(1), atomic.LoadInt64(&received), "Event should not be received after unsubscribe")
}

// TestAsyncEventBus_PublishRacingClose_NeverSilentlyDrops is the regression
// test for ISS-243 / TASK-245: Publish must never report success (nil) for an
// event the consumer will not deliver.
//
// The old implementation checked closed.Load() and then did a non-blocking
// send, with no synchronization against Close. A publish that passed the check
// just before Close ran could land its event in the buffer AFTER processLoop
// had drained and exited: Publish returned nil but the event was never
// delivered — a silent drop. The fix serializes the closed-check + send against
// Close (async_bus.mu), which makes "every successful publish is delivered"
// a hard invariant: any publish that saw closed==false happened-before done was
// closed, so Close's drain (waited on by wg.Wait) is guaranteed to pick it up.
//
// The race window is nanoseconds wide, so a single run detects a reintroduced
// check-then-act only probabilistically (measured: the unfixed code trips this
// test across iterations; the fixed code never does, since the invariant is
// exact rather than statistical).
func TestAsyncEventBus_PublishRacingClose_NeverSilentlyDrops(t *testing.T) {
	const iters = 200
	for iter := 0; iter < iters; iter++ {
		bus := NewAsyncEventBus(4)
		var consumed, success int64
		bus.SubscribeAll(func(e events.Event) error {
			atomic.AddInt64(&consumed, 1)
			return nil
		})

		// A background streamer keeps publishing right across the Close window.
		stop := make(chan struct{})
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				if bus.Publish(events.NewBaseEvent("e", "agg", []byte(`{}`))) == nil {
					atomic.AddInt64(&success, 1)
				}
			}
		}()

		// A burst from this goroutine too, then Close while both are mid-flight.
		for i := 0; i < 5000; i++ {
			if bus.Publish(events.NewBaseEvent("e", "agg", []byte(`{}`))) == nil {
				atomic.AddInt64(&success, 1)
			}
		}
		bus.Close()
		close(stop)

		got, want := atomic.LoadInt64(&consumed), atomic.LoadInt64(&success)
		if got != want {
			t.Fatalf("iteration %d: consumed=%d but success=%d — an event was accepted and silently dropped", iter, got, want)
		}
	}
}

func TestAsyncEventBus_ChannelFull(t *testing.T) {
	// Subscribe a blocking handler so the consumer goroutine cannot drain
	// the buffer out from under the test. Without this, Publish would race
	// with processLoop and the test would be flaky.
	bus := NewAsyncEventBus(2)
	defer bus.Close()

	block := make(chan struct{})
	bus.Subscribe("e", func(e events.Event) error {
		<-block
		return nil
	})

	// Give the consumer a moment to receive the first event and block on it,
	// so the channel is now empty again and ready to fill.
	_ = bus.Publish(events.NewBaseEvent("e", "1", nil))
	time.Sleep(10 * time.Millisecond)

	// Now the consumer is blocked inside the handler. The channel is empty.
	// Two more publishes fill the buffer; the third must fail with full.
	require.NoError(t, bus.Publish(events.NewBaseEvent("e", "2", nil)))
	require.NoError(t, bus.Publish(events.NewBaseEvent("e", "3", nil)))
	require.ErrorIs(t, bus.Publish(events.NewBaseEvent("e", "4", nil)),
		events.ErrEventBusFull,
		"4th publish must fail because consumer is blocked and buffer is size 2")

	close(block)
}

func TestCompositeEventBus_SyncBlocksAsync(t *testing.T) {
	bus := NewCompositeEventBus()
	defer bus.AsyncBus.Close()

	var syncCalled bool
	bus.SyncBus.SubscribeAll(func(e events.Event) error {
		syncCalled = true
		return errors.New("sync error")
	})

	var asyncCalled bool
	bus.AsyncBus.Subscribe("test", func(e events.Event) error {
		asyncCalled = true
		return nil
	})

	err := bus.Publish(events.NewBaseEvent("test", "agg", nil))
	require.Error(t, err)
	require.True(t, syncCalled)
	require.False(t, asyncCalled)
}
