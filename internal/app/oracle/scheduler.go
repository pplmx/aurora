package oracle

import (
	"context"
	"sync"
	"time"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/logger"
)

// sourceRepo abstracts the read surface the scheduler needs: enumerate
// candidate sources and look up the latest recorded data per source so we can
// seed a last-fetch time and avoid re-fetching everything immediately after a
// restart. Any Repository implementation (SQLite, in-memory) or a fake in
// tests satisfies it.
type sourceRepo interface {
	ListSources() ([]*oracle.DataSource, error)
	GetLatestData(sourceID string) (*oracle.OracleData, error)
}

// Scheduler periodically fetches enabled data sources on their configured
// Interval. The Interval field was stored and exposed through the API/CLI but
// never acted on — this closes that gap (v1.15 Oracle Scheduled Fetching).
//
// It is concurrency-safe: the last-fetch bookkeeping is guarded by a mutex so
// Run/stop and the fetch callback never race.
type Scheduler struct {
	repo       sourceRepo
	execute    func(sourceID string) error
	checkEvery time.Duration
	now        func() time.Time

	mu        sync.Mutex
	lastFetch map[string]time.Time
	seeded    bool
}

// NewScheduler returns a scheduler that polls the repository on checkEvery and
// fetches each enabled source whose interval has elapsed since its last
// successful fetch. now is injectable for deterministic tests; nil falls back
// to time.Now.
func NewScheduler(repo sourceRepo, execute func(sourceID string) error, checkEvery time.Duration, now func() time.Time) *Scheduler {
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
	s.seed()
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
	s.seed()
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

// seed initialises lastFetch from each source's latest recorded data so a
// freshly started process does not immediately re-fetch a source whose data is
// still inside its refresh interval. It runs exactly once (idempotent) and
// tolerates sources whose data cannot be read.
func (s *Scheduler) seed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seeded {
		return
	}
	s.seeded = true

	sources, err := s.repo.ListSources()
	if err != nil {
		logger.Warn().Err(err).Msg("oracle scheduler: failed to list sources on seed")
		return
	}
	for _, src := range sources {
		d, err := s.repo.GetLatestData(src.ID)
		if err != nil {
			logger.Warn().Err(err).Str("source_id", src.ID).Msg("oracle scheduler: failed to read latest data on seed")
			continue
		}
		if d == nil || d.Timestamp <= 0 {
			continue
		}
		s.lastFetch[src.ID] = time.Unix(d.Timestamp, 0)
	}
}
