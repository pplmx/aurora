package oracle

import (
	"context"
	"fmt"
	"sort"
	"strings"
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

// maxBackoff caps the exponential backoff applied to a persistently-failing
// source (v1.17). Without a cap a long outage would drive the retry gap to
// unbounded lengths.
const maxBackoff = 5 * time.Minute

// backoffBase is the first retry gap after one failure (matches the default
// check cadence).
const backoffBase = time.Second

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

	mu          sync.Mutex
	lastFetch   map[string]time.Time
	seeded      bool
	stats       map[string]*SourceStat
	failStreak  map[string]int
	nextAttempt map[string]time.Time
}

// SourceStat is the cumulative fetch-health counter for one data source. It is
// the operator-visible feed-health surface (v1.16): attempts / successes /
// failures since process start, plus the last error and last-success time.
type SourceStat struct {
	SourceID      string `json:"source_id"`
	Attempts      uint64 `json:"attempts"`
	Successes     uint64 `json:"successes"`
	Failures      uint64 `json:"failures"`
	LastSuccessAt int64  `json:"last_success_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
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
		repo:        repo,
		execute:     execute,
		checkEvery:  checkEvery,
		now:         now,
		lastFetch:   make(map[string]time.Time),
		stats:       make(map[string]*SourceStat),
		failStreak:  make(map[string]int),
		nextAttempt: make(map[string]time.Time),
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
		if due && now.Before(s.nextAttempt[src.ID]) {
			due = false
		}
		s.mu.Unlock()
		if !due {
			continue
		}

		s.mu.Lock()
		st := s.stat(src.ID)
		st.Attempts++
		s.mu.Unlock()

		if err := s.execute(src.ID); err != nil {
			s.mu.Lock()
			st.Failures++
			st.LastError = err.Error()
			s.failStreak[src.ID]++
			s.nextAttempt[src.ID] = now.Add(backoff(s.failStreak[src.ID]))
			s.mu.Unlock()
			logger.Warn().Err(err).Str("source_id", src.ID).Msg("oracle scheduler: fetch failed")
			continue
		}

		s.mu.Lock()
		st.Successes++
		st.LastSuccessAt = now.Unix()
		s.lastFetch[src.ID] = now
		delete(s.failStreak, src.ID)
		delete(s.nextAttempt, src.ID)
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

// stat returns the SourceStat for id, creating it if absent. Caller must hold
// the mutex.
func (s *Scheduler) stat(id string) *SourceStat {
	st, ok := s.stats[id]
	if !ok {
		st = &SourceStat{SourceID: id}
		s.stats[id] = st
	}
	return st
}

// Stats returns a copy of the per-source fetch-health statistics, sorted by
// source id for deterministic output.
func (s *Scheduler) Stats() []SourceStat {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SourceStat, 0, len(s.stats))
	for _, st := range s.stats {
		c := *st
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SourceID < out[j].SourceID })
	return out
}

// PrometheusText renders the per-source fetch-health stats in Prometheus text
// exposition format for an operator metrics endpoint (/metrics/oracle).
func (s *Scheduler) PrometheusText() string {
	stats := s.Stats()
	var b strings.Builder
	fmt.Fprintln(&b, "# TYPE oracle_fetch_attempts_total counter")
	fmt.Fprintln(&b, "# TYPE oracle_fetch_successes_total counter")
	fmt.Fprintln(&b, "# TYPE oracle_fetch_failures_total counter")
	for _, st := range stats {
		fmt.Fprintf(&b, "oracle_fetch_attempts_total{source=%q} %d\n", st.SourceID, st.Attempts)
		fmt.Fprintf(&b, "oracle_fetch_successes_total{source=%q} %d\n", st.SourceID, st.Successes)
		fmt.Fprintf(&b, "oracle_fetch_failures_total{source=%q} %d\n", st.SourceID, st.Failures)
	}
	return b.String()
}

// backoff returns the retry gap for the given consecutive-failure count:
// 1s, 2s, 4s, ... capped at maxBackoff. count==1 returns backoffBase.
func backoff(streak int) time.Duration {
	if streak <= 1 {
		return backoffBase
	}
	d := backoffBase
	for i := 1; i < streak; i++ {
		d *= 2
		if d >= maxBackoff {
			return maxBackoff
		}
	}
	return d
}
