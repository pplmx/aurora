package oracle

import (
	"errors"
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
	// A failed fetch must not be marked done, so it is retried next pass even
	// though the interval has not elapsed.
	s.pass()
	if attempts != 2 {
		t.Fatalf("expected failed fetch to be retried next pass, got %d attempts", attempts)
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
