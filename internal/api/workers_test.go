package api

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkerGroupStopsInReverseOrderExactlyOnce(t *testing.T) {
	var group workerGroup
	var mu sync.Mutex
	var stopped []string
	record := func(name string) func() {
		return func() {
			mu.Lock()
			stopped = append(stopped, name)
			mu.Unlock()
		}
	}

	stopScheduler := group.add(record("scheduler"))
	group.add(record("outbox"))

	stopScheduler()
	stopScheduler()
	group.close()
	group.close()

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"scheduler", "outbox"}, stopped)
}

func TestWorkerGroupConcurrentCloseWaitsForWorkers(t *testing.T) {
	var group workerGroup
	started := make(chan struct{})
	release := make(chan struct{})
	group.add(func() {
		close(started)
		<-release
	})

	first := make(chan struct{})
	go func() {
		group.close()
		close(first)
	}()
	require.Eventually(t, func() bool {
		select {
		case <-started:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)

	second := make(chan struct{})
	go func() {
		group.close()
		close(second)
	}()

	select {
	case <-second:
		t.Fatal("a concurrent Close returned before the running stop function finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	for name, done := range map[string]<-chan struct{}{"first": first, "second": second} {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("%s Close did not return after the worker stopped", name)
		}
	}
}

func TestWorkerGroupStopsLateRegistration(t *testing.T) {
	var group workerGroup
	group.close()

	var mu sync.Mutex
	called := 0
	stop := group.add(func() {
		mu.Lock()
		called++
		mu.Unlock()
	})
	stop()

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, called, "a worker registered after Close must stop immediately and only once")
}
