package oracle

import (
	"errors"
	"strings"
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
	s := NewScheduler(repo, func(id string) error { fetched = append(fetched, id); return nil }, time.Second, func() time.Time { return now })

	// First pass: everything enabled with interval>0 is due.
	s.pass()
	if got := len(fetched); got != 2 {
		t.Fatalf("first pass fetched %d sources, want 2 (a,b): %v", got, fetched)
	}

	// Advance 59s: a(60s) not due yet, b(20s) is due.
	now = now.Add(59 * time.Second)
	before := len(fetched)
	s.pass()
	// only b refetched
	if len(fetched) != before+1 || fetched[len(fetched)-1] != "b" {
		t.Fatalf("after 59s expected only b refetched, fetched=%v", fetched[before:])
	}

	// Advance 2s more (61s total): a now due.
	now = now.Add(2 * time.Second)
	before = len(fetched)
	s.pass()
	if len(fetched) != before+1 || fetched[len(fetched)-1] != "a" {
		t.Fatalf("after 61s expected only a refetched, fetched=%v", fetched[before:])
	}
}

func TestScheduler_PassRetriesFailed(t *testing.T) {
	now := time.Unix(1000, 0)
	repo := &fakeSourceRepo{sources: []*oracle.DataSource{
		{ID: "a", Enabled: true, Interval: 60},
	}}
	attempts := 0
	s := NewScheduler(repo, func(id string) error {
		attempts++
		return errors.New("boom")
	}, time.Second, func() time.Time { return now })

	s.pass()
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
	// Backoff (v1.17): a failed fetch is suppressed during its backoff window
	// (1s after the first failure), so the immediate next pass must NOT retry.
	s.pass()
	if attempts != 1 {
		t.Fatalf("expected failed fetch to be suppressed during backoff, got %d attempts", attempts)
	}
	// After the backoff window elapses the source is retried.
	now = now.Add(2 * time.Second)
	s.pass()
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
	s := NewScheduler(repo, func(id string) error { fetched = append(fetched, id); return nil }, time.Second, func() time.Time { return now })

	s.pass()
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
	s := NewScheduler(repo, func(id string) error {
		if id == "bad" {
			return errors.New("boom")
		}
		return nil
	}, time.Second, func() time.Time { return now })

	s.pass()
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
	s.pass()
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
	s := NewScheduler(repo, func(id string) error { return nil }, time.Second, func() time.Time { return now })
	s.pass()

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
