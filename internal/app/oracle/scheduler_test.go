package oracle

import (
	"errors"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

type fakeSourceRepo struct {
	sources []*oracle.DataSource
}

func (f *fakeSourceRepo) ListSources() ([]*oracle.DataSource, error) { return f.sources, nil }

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
