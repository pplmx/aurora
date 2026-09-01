package oracle

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

type fakeSourceRepo struct {
	sources []*oracle.DataSource
	data    map[string]*oracle.OracleData
}

func (f *fakeSourceRepo) ListSources() ([]*oracle.DataSource, error) { return f.sources, nil }

func (f *fakeSourceRepo) GetLatestData(sourceID string) (*oracle.OracleData, error) {
	return f.data[sourceID], nil
}

func TestScheduler_PassFetchesDueSourcesOnly(t *testing.T) {
	now := time.Unix(1000, 0)
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "a", Enabled: true, Interval: 60},
		{ID: "b", Enabled: true, Interval: 20},
		{ID: "disabled", Enabled: false, Interval: 60},
		{ID: "zero", Enabled: true, Interval: 0},
	}}
	var fetched []string
	s := NewScheduler(repo, func(_ context.Context, id string) error { fetched = append(fetched, id); return nil }, time.Second, func() time.Time { return now })

	// First pass: everything enabled with interval>0 is due.
	s.pass(context.Background())
	if got := len(fetched); got != 2 {
		t.Fatalf("first pass fetched %d sources, want 2 (a,b): %v", got, fetched)
	}

	// Advance 59s: a(60s) not due yet, b(20s) is due.
	now = now.Add(59 * time.Second)
	before := len(fetched)
	s.pass(context.Background())
	// only b refetched
	if len(fetched) != before+1 || fetched[len(fetched)-1] != "b" {
		t.Fatalf("after 59s expected only b refetched, fetched=%v", fetched[before:])
	}

	// Advance 2s more (61s total): a now due.
	now = now.Add(2 * time.Second)
	before = len(fetched)
	s.pass(context.Background())
	if len(fetched) != before+1 || fetched[len(fetched)-1] != "a" {
		t.Fatalf("after 61s expected only a refetched, fetched=%v", fetched[before:])
	}
}

// TestScheduler_Run_FetchesAndStopsOnCancel exercises the production entry
// point (Scheduler.Run), previously the package's only uncovered branch. The
// scheduler's sources use whole-second intervals (min 1s), so the initial pass
// yields one fetch and a full interval-elapse is needed for a second
// ticker-driven fetch. The test asserts Run performs its initial fetch, keeps
// polling on its ticker (a second fetch arrives after the interval elapses),
// and returns promptly (no goroutine leak) once the context is cancelled.
func TestScheduler_Run_FetchesAndStopsOnCancel(t *testing.T) {
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "a", Enabled: true, Interval: 1},
	}}
	var calls int64
	s := NewScheduler(repo, func(_ context.Context, id string) error {
		atomic.AddInt64(&calls, 1)
		return nil
	}, 10*time.Millisecond, time.Now)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()

	// Wait for the initial pass (call 1) plus one ticker-driven refetch once
	// the 1s interval elapses (call 2). The deadline is generous to stay
	// reliable under -race.
	deadline := time.Now().Add(5 * time.Second)
	for atomic.LoadInt64(&calls) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	callsAtCancel := atomic.LoadInt64(&calls)
	if callsAtCancel < 2 {
		t.Fatalf("expected Run to fetch at least twice (initial + ticker-driven refetch), got %d", callsAtCancel)
	}

	// Cancelling must make Run return.
	cancel()
	select {
	case <-done:
		// Returned as expected.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation (goroutine leak)")
	}
}

// TestScheduler_Run_StoppedContextShutsDownImmediately verifies that a
// scheduler started with an already-cancelled context does not loop: Run
// performs its single initial pass (seed + first fetch) and then returns
// immediately without any ticker polling. This pins the fact that a cancelled
// context stops the loop, not that it skips the synchronous initial pass.
func TestScheduler_Run_StoppedContextShutsDownImmediately(t *testing.T) {
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "a", Enabled: true, Interval: 60},
	}}
	var calls int64
	s := NewScheduler(repo, func(_ context.Context, id string) error {
		atomic.AddInt64(&calls, 1)
		return nil
	}, 10*time.Millisecond, time.Now)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return for a pre-cancelled context")
	}
	if atomic.LoadInt64(&calls) != 1 {
		t.Fatalf("expected exactly the initial-pass fetch for pre-cancelled context, got %d", calls)
	}
}

func TestScheduler_PassRetriesFailed(t *testing.T) {
	now := time.Unix(1000, 0)
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "a", Enabled: true, Interval: 60},
	}}
	attempts := 0
	s := NewScheduler(repo, func(_ context.Context, id string) error {
		attempts++
		return errors.New("boom")
	}, time.Second, func() time.Time { return now })

	s.pass(context.Background())
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
	// Backoff (v1.17): a failed fetch is suppressed during its backoff window
	// (1s after the first failure), so the immediate next pass must NOT retry.
	s.pass(context.Background())
	if attempts != 1 {
		t.Fatalf("expected failed fetch to be suppressed during backoff, got %d attempts", attempts)
	}
	// After the backoff window elapses the source is retried.
	now = now.Add(2 * time.Second)
	s.pass(context.Background())
	if attempts != 2 {
		t.Fatalf("expected failed fetch to be retried after backoff, got %d attempts", attempts)
	}
}
func TestScheduler_SeedsLastFetchFromRepo(t *testing.T) {
	now := time.Unix(2000, 0)
	// Source "fresh" has data recorded at 1990 (10s ago, interval 60s) -> not
	// due on start. Source "stale" has data at 1800 (200s ago, interval 60s)
	// -> overdue, fetched. Source "none" has no data -> fetched.
	repo := &fakeSourceRepo{
		sources: []*oracle.DataSource{
			{ID: "fresh", Enabled: true, Interval: 60},
			{ID: "stale", Enabled: true, Interval: 60},
			{ID: "none", Enabled: true, Interval: 60},
		},
		data: map[string]*oracle.OracleData{
			"fresh": {SourceID: "fresh", Timestamp: 1990},
			"stale": {SourceID: "stale", Timestamp: 1800},
		},
	}
	var fetched []string
	s := NewScheduler(repo, func(_ context.Context, id string) error { fetched = append(fetched, id); return nil }, time.Second, func() time.Time { return now })

	s.pass(context.Background())
	if len(fetched) != 2 {
		t.Fatalf("expected fresh (not yet due) to be skipped and stale+none fetched, got %v", fetched)
	}
	got := map[string]bool{}
	for _, id := range fetched {
		got[id] = true
	}
	if !got["stale"] || !got["none"] {
		t.Fatalf("expected stale and none to be fetched, got %v", fetched)
	}
	if got["fresh"] {
		t.Fatalf("fresh source should NOT be re-fetched (data still inside interval), got %v", fetched)
	}
}
func TestScheduler_StatsTracksSuccessAndFailure(t *testing.T) {
	now := time.Unix(1000, 0)
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "ok", Enabled: true, Interval: 60},
		{ID: "bad", Enabled: true, Interval: 60},
	}}
	s := NewScheduler(repo, func(_ context.Context, id string) error {
		if id == "bad" {
			return errors.New("boom")
		}
		return nil
	}, time.Second, func() time.Time { return now })

	s.pass(context.Background())
	stats := map[string]SourceStat{}
	for _, st := range s.Stats() {
		stats[st.SourceID] = st
	}

	ok, hasOK := stats["ok"]
	if !hasOK || ok.Attempts != 1 || ok.Successes != 1 || ok.Failures != 0 || ok.LastSuccessAt != now.Unix() {
		t.Fatalf("unexpected ok stat: %+v", ok)
	}
	bad, hasBad := stats["bad"]
	if !hasBad || bad.Attempts != 1 || bad.Successes != 0 || bad.Failures != 1 || bad.LastError != "boom" {
		t.Fatalf("unexpected bad stat: %+v", bad)
	}

	// After the backoff window elapses the failed fetch is retried, so
	// attempts/failures climb.
	now = now.Add(2 * time.Second)
	s.pass(context.Background())
	bad2 := map[string]SourceStat{}
	for _, st := range s.Stats() {
		bad2[st.SourceID] = st
	}
	if b := bad2["bad"]; b.Attempts != 2 || b.Failures != 2 {
		t.Fatalf("expected bad attempts=2 failures=2 after retry, got %+v", b)
	}
}

func TestScheduler_PrometheusText(t *testing.T) {
	now := time.Unix(1000, 0)
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "a", Enabled: true, Interval: 60},
	}}
	s := NewScheduler(repo, func(_ context.Context, id string) error { return nil }, time.Second, func() time.Time { return now })
	s.pass(context.Background())

	out := s.PrometheusText()
	for _, want := range []string{
		`oracle_fetch_attempts_total{source="a"} 1`,
		`oracle_fetch_successes_total{source="a"} 1`,
		`oracle_fetch_failures_total{source="a"} 0`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("PrometheusText missing %q:\n%s", want, out)
		}
	}
}
func TestBackoffEscalatesAndCaps(t *testing.T) {
	cases := []struct {
		streak int
		want   time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
	}
	for _, c := range cases {
		if got := backoff(c.streak); got != c.want {
			t.Errorf("backoff(%d)=%s, want %s", c.streak, got, c.want)
		}
	}
	// The retry gap must never exceed maxBackoff no matter how long the outage.
	for streak := 6; streak <= 20; streak++ {
		if d := backoff(streak); d > maxBackoff {
			t.Errorf("backoff(%d)=%s exceeds maxBackoff %s", streak, d, maxBackoff)
		}
	}
}

// TestScheduler_CancelInterruptsInFlightFetch pins the TASK-134/ISS-127 fix:
// the execute closure must receive the scheduler's ctx so an in-flight fetch
// is interrupted on shutdown (previously the fetch used a background-context
// HTTP request and blocked up to the client timeout, stalling srv.Close()).
// It also verifies pass() does not start further sources once cancelled.
func TestScheduler_CancelInterruptsInFlightFetch(t *testing.T) {
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "a", Enabled: true, Interval: 60},
		{ID: "b", Enabled: true, Interval: 60},
	}}
	var calls int64
	started := make(chan struct{})
	gotCancel := make(chan struct{})
	s := NewScheduler(repo, func(ctx context.Context, id string) error {
		if atomic.AddInt64(&calls, 1) == 1 {
			close(started)
			<-ctx.Done() // simulate an HTTP fetch that only returns on cancel
			close(gotCancel)
		}
		return ctx.Err()
	}, time.Second, func() time.Time { return time.Unix(1000, 0) })

	ctx, cancelIt := context.WithCancel(context.Background())
	go s.Run(ctx)

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler never started the first fetch")
	}

	cancelIt() // shutdown while the fetch is in flight

	select {
	case <-gotCancel:
		// The in-flight fetch's context was cancelled — it did not have to
		// wait out its full timeout.
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight fetch context was not cancelled on shutdown")
	}

	// Give pass() a moment to reach its per-source cancellation check, then
	// assert no second source was fetched after cancel.
	deadline := time.Now().Add(time.Second)
	for atomic.LoadInt64(&calls) != 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if atomic.LoadInt64(&calls) != 1 {
		t.Fatalf("expected only the in-flight source to be fetched after cancel, got %d calls", atomic.LoadInt64(&calls))
	}
}

// TestScheduler_ReconcileRemovesDeletedSource pins TASK-242/ISS-240: deleting a
// source at runtime must drop its per-source bookkeeping from the scheduler —
// otherwise /oracle/health and /metrics/oracle advertise sources that no longer
// exist and the maps (lastFetch, stats, failStreak, nextAttempt) grow without
// bound across create/delete cycles.
func TestScheduler_ReconcileRemovesDeletedSource(t *testing.T) {
	now := time.Unix(1000, 0)
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "keep", Enabled: true, Interval: 60},
		{ID: "doomed", Enabled: true, Interval: 60},
	}}
	s := NewScheduler(repo, func(_ context.Context, id string) error { return nil }, time.Second, func() time.Time { return now })
	// "doomed" fails on every attempt, so it also accrues failStreak/nextAttempt
	// state alongside its stats entry.
	s.execute = func(_ context.Context, id string) error {
		if id == "doomed" {
			return errors.New("boom")
		}
		return nil
	}

	s.pass(context.Background())
	if got := len(s.Stats()); got != 2 {
		t.Fatalf("after first pass Stats() has %d entries, want 2", got)
	}
	s.mu.Lock()
	_, hadStreak := s.failStreak["doomed"]
	_, hadNext := s.nextAttempt["doomed"]
	s.mu.Unlock()
	if !hadStreak || !hadNext {
		t.Fatal("doomed source should have accrued failure/backoff state before deletion")
	}

	// Delete "doomed" from the repo; the next pass must prune it everywhere.
	repo.sources = repo.sources[:1]
	s.pass(context.Background())

	stats := s.Stats()
	if len(stats) != 1 || stats[0].SourceID != "keep" {
		t.Fatalf("after delete Stats() must contain only the live source, got %v", stats)
	}
	s.mu.Lock()
	_, hasStats := s.stats["doomed"]
	_, hasStreak := s.failStreak["doomed"]
	_, hasNext := s.nextAttempt["doomed"]
	_, hasLast := s.lastFetch["doomed"]
	s.mu.Unlock()
	if hasStats || hasStreak || hasNext || hasLast {
		t.Fatal("deleted source must be fully pruned from scheduler bookkeeping")
	}
}
