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

func TestAsyncEventBus_ChannelFull(t *testing.T) {
	bus := NewAsyncEventBus(2)
	defer bus.Close()

	_ = bus.Publish(events.NewBaseEvent("e", "1", nil))
	_ = bus.Publish(events.NewBaseEvent("e", "2", nil))

	err := bus.Publish(events.NewBaseEvent("e", "3", nil))
	require.Error(t, err)
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
