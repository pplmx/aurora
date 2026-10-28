package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/stretchr/testify/require"
)

// blockingOracleRepo holds the scheduler's first repository call open so a
// lifecycle test can distinguish "Close returned" from "Close waited for the
// worker to leave the repository".
type blockingOracleRepo struct {
	*oracle.InmemRepo
	entered     chan struct{}
	release     chan struct{}
	enteredOnce sync.Once
	releaseOnce sync.Once
}

func newBlockingOracleRepo() *blockingOracleRepo {
	return &blockingOracleRepo{
		InmemRepo: oracle.NewInmemRepo(),
		entered:   make(chan struct{}),
		release:   make(chan struct{}),
	}
}

func (r *blockingOracleRepo) ListSources(limit, offset int) ([]*oracle.DataSource, error) {
	r.enteredOnce.Do(func() {
		close(r.entered)
		<-r.release
	})
	return r.InmemRepo.ListSources(limit, offset)
}

func (r *blockingOracleRepo) unblock() {
	r.releaseOnce.Do(func() {
		close(r.release)
	})
}

// TestServerCloseWaitsForOracleScheduler covers the non-graceful API failure
// path: cmd/api calls Server.Close directly before logger.Fatal, which skips
// deferred stop functions. Close therefore owns the started workers and must
// wait for the scheduler to leave its repository before releasing SQLite.
func TestServerCloseWaitsForOracleScheduler(t *testing.T) {
	repo := newBlockingOracleRepo()
	t.Cleanup(repo.unblock)

	srv := &Server{
		oracleRuntime: oracleapp.NewRuntime(repo, nil, nil),
	}
	srv.StartOracleScheduler(context.Background(), time.Hour)

	select {
	case <-repo.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not enter the repository")
	}

	closed := make(chan error, 1)
	go func() {
		closed <- srv.Close()
	}()

	select {
	case err := <-closed:
		t.Fatalf("Server.Close returned while the scheduler was still in the repository: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	repo.unblock()
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server.Close did not finish after the scheduler left the repository")
	}
}
