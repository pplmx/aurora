package api

import "sync"

// workerGroup owns the stop functions for background workers started by a
// Server. Close stops them in reverse start order before releasing storage,
// including the non-graceful cmd/api path that cannot run deferred stops.
//
// The returned stop function is idempotent, so callers may stop a worker
// explicitly and Close may still safely stop the whole group. Concurrent Close
// calls wait for the first group stop to finish before returning.
type workerGroup struct {
	mu     sync.Mutex
	closed bool
	done   chan struct{}
	stops  []func()
}

func (g *workerGroup) add(stop func()) func() {
	if stop == nil {
		stop = func() {}
	}
	stop = sync.OnceFunc(stop)

	g.mu.Lock()
	if !g.closed {
		g.stops = append(g.stops, stop)
		g.mu.Unlock()
		return stop
	}
	g.mu.Unlock()

	// A worker registered after Close must not outlive the storage it reads.
	stop()
	return stop
}

func (g *workerGroup) close() {
	g.mu.Lock()
	if g.closed {
		done := g.done
		g.mu.Unlock()
		if done != nil {
			<-done
		}
		return
	}
	g.closed = true
	g.done = make(chan struct{})
	done := g.done
	stops := g.stops
	g.stops = nil
	g.mu.Unlock()

	defer close(done)
	for i := len(stops) - 1; i >= 0; i-- {
		stops[i]()
	}
}
