package oracle

import (
	"context"
	"sync"
	"time"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/logger"
)

// sourceLister abstracts listing candidate sources so the scheduler can be
// driven by any Repository implementation (SQLite, in-memory) or a fake in
// tests.
type sourceLister interface {
	ListSources() ([]*oracle.DataSource, error)
}

// Scheduler periodically fetches enabled data sources on their configured
// Interval. The Interval field was stored and exposed through the API/CLI but
// never acted on — this closes that gap (v1.15 Oracle Scheduled Fetching).
//
// It is concurrency-safe: the last-fetch bookkeeping is guarded by a mutex so
// Run/stop and the fetch callback never race.
type Scheduler struct {
	repo       sourceLister
	execute    func(sourceID string) error
	checkEvery time.Duration
	now        func() time.Time

	mu        sync.Mutex
	lastFetch map[string]time.Time
}

// NewScheduler returns a scheduler that polls the repository on checkEvery and
// fetches each enabled source whose interval has elapsed since its last
// successful fetch. now is injectable for deterministic tests; nil falls back
// to time.Now.
func NewScheduler(repo sourceLister, execute func(sourceID string) error, checkEvery time.Duration, now func() time.Time) *Scheduler {
	if now == nil {
		now = time.Now
	}
	if checkEvery <= 0 {
		checkEvery = time.Second
	}
	return &Scheduler{
		repo:       repo,
		execute:    execute,
		checkEvery: checkEvery,
		now:        now,
		lastFetch:  make(map[string]time.Time),
	}
}

// Run blocks, polling the repository until ctx is cancelled. It performs an
// initial pass immediately, then re-checks on checkEvery.
func (s *Scheduler) Run(ctx context.Context) {
	s.pass()
	t := time.NewTicker(s.checkEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.pass()
		}
	}
}

// pass checks every enabled source and fetches the ones whose interval has
// elapsed. A failed fetch is not marked as done, so the next pass retries it.
func (s *Scheduler) pass() {
	sources, err := s.repo.ListSources()
	if err != nil {
		logger.Warn().Err(err).Msg("oracle scheduler: failed to list sources")
		return
	}

	for _, src := range sources {
		if !src.Enabled || src.Interval <= 0 {
			continue
		}
		interval := time.Duration(src.Interval) * time.Second
		now := s.now()

		s.mu.Lock()
		last, ok := s.lastFetch[src.ID]
		due := !ok || now.Sub(last) >= interval
		s.mu.Unlock()
		if !due {
			continue
		}

		if err := s.execute(src.ID); err != nil {
			logger.Warn().Err(err).Str("source_id", src.ID).Msg("oracle scheduler: fetch failed")
			continue
		}

		s.mu.Lock()
		s.lastFetch[src.ID] = now
		s.mu.Unlock()
	}
}
