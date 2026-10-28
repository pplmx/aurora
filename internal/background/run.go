// Package background runs cancellable work with a stop function that does not
// return until the work has observed cancellation and exited.
package background

import (
	"context"
	"sync"
)

// Start runs fn in a goroutine. The returned stop function cancels fn's
// context and waits for it to return. Repeated calls to stop are safe and wait
// for the same completion, so callers can release storage handles only after
// stop returns.
func Start(ctx context.Context, fn func(context.Context)) (stop func()) {
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(runCtx)
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			<-done
		})
	}
}
