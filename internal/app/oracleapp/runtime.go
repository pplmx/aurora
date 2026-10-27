package oracleapp

import (
	"context"
	"sync"
	"time"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

// Runtime owns the optional oracle scheduler and the fetch use case it drives.
// The HTTP layer reads its health/metrics; the desktop shares the same instance
// through api.Server, so both surfaces report the scheduler's real counters.
type Runtime struct {
	mu      sync.Mutex
	repo    oracle.Repository
	fetcher FetcherInterface
	chain   ChainInterface

	scheduler *Scheduler
	fetch     *FetchDataUseCase
	cancel    context.CancelFunc
}

func NewRuntime(repo oracle.Repository, fetcher FetcherInterface, chain ChainInterface) *Runtime {
	return &Runtime{repo: repo, fetcher: fetcher, chain: chain}
}

// Start builds and runs the scheduler. It replaces a previous run, so a caller
// that restarts the scheduler does not leak the old goroutine.
func (r *Runtime) Start(ctx context.Context, checkEvery time.Duration) (stop func()) {
	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.repo == nil {
		r.mu.Unlock()
		return func() {}
	}
	fetch := NewFetchDataUseCaseWithDeps(r.repo, r.fetcher)
	fetch.SetChain(r.chain)
	runner := func(ctx context.Context, sourceID string) error {
		_, err := fetch.ExecuteContext(ctx, &FetchDataRequest{SourceID: sourceID})
		return err
	}
	sched := NewScheduler(r.repo, runner, checkEvery, nil)
	schedCtx, cancel := context.WithCancel(ctx)
	r.scheduler = sched
	r.fetch = fetch
	r.cancel = cancel
	r.mu.Unlock()

	go sched.Run(schedCtx)
	return cancel
}

// Fetch exposes the scheduler's use case for wiring tests; nil before Start.
func (r *Runtime) Fetch() *FetchDataUseCase {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fetch
}

// Stats returns the scheduler counters, or nil before Start / without a repo.
func (r *Runtime) Stats() []SourceStat {
	r.mu.Lock()
	sched := r.scheduler
	r.mu.Unlock()
	if sched == nil {
		return nil
	}
	return sched.Stats()
}

// PrometheusText returns the scheduler's metrics, empty before Start.
func (r *Runtime) PrometheusText() string {
	r.mu.Lock()
	sched := r.scheduler
	r.mu.Unlock()
	if sched == nil {
		return ""
	}
	return sched.PrometheusText()
}
