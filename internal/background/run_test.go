package background

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStartWaitsForRunToExit(t *testing.T) {
	started := make(chan struct{})
	cleanupStarted := make(chan struct{})
	release := make(chan struct{})

	stop := Start(context.Background(), func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(cleanupStarted)
		<-release
	})
	<-started

	stopped := make(chan struct{})
	go func() {
		stop()
		close(stopped)
	}()

	<-cleanupStarted
	select {
	case <-stopped:
		t.Fatal("stop returned while the background work was still using its store")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop did not return after the background work exited")
	}
}

func TestStartStopIsIdempotent(t *testing.T) {
	exited := make(chan struct{})
	stop := Start(context.Background(), func(ctx context.Context) {
		<-ctx.Done()
		close(exited)
	})

	stop()
	require.Eventually(t, func() bool {
		select {
		case <-exited:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	stop()
}
