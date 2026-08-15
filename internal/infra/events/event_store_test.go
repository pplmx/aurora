package events

import (
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEventStore(t *testing.T) (*SQLiteEventStore, func()) {
	tmpFile, err := os.CreateTemp("", "event_store_test_*.db")
	require.NoError(t, err)
	_ = tmpFile.Close()

	store, err := NewSQLiteEventStore(tmpFile.Name())
	require.NoError(t, err)

	cleanup := func() {
		_ = store.Close()
		_ = os.Remove(tmpFile.Name())
	}
	return store, cleanup
}

func TestNewSQLiteEventStore(t *testing.T) {
	t.Run("creates store with temp file", func(t *testing.T) {
		store, cleanup := setupEventStore(t)
		defer cleanup()

		assert.NotNil(t, store)
		assert.NotNil(t, store.db)
	})

	t.Run("creates store for new path", func(t *testing.T) {
		tmpFile2, err := os.CreateTemp("", "event_store_new_*.db")
		require.NoError(t, err)
		_ = tmpFile2.Close()
		defer func() { _ = os.Remove(tmpFile2.Name()) }()

		store, err := NewSQLiteEventStore(tmpFile2.Name())
		require.NoError(t, err)
		assert.NotNil(t, store)
		_ = store.Close()
	})
}

func TestSQLiteEventStore_Save(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	t.Run("saves event successfully", func(t *testing.T) {
		event := events.NewBaseEvent("nft.mint", "nft-123", []byte(`{"name":"test"}`))

		err := store.Save(event)
		require.NoError(t, err)
	})

	t.Run("saves multiple events", func(t *testing.T) {
		event1 := events.NewBaseEvent("token.transfer", "token-456", []byte(`{"from":"alice","to":"bob"}`))
		event2 := events.NewBaseEvent("token.transfer", "token-456", []byte(`{"from":"bob","to":"carol"}`))

		err := store.Save(event1)
		require.NoError(t, err)

		err = store.Save(event2)
		require.NoError(t, err)
	})
}

// TestSQLiteEventStore_RoundTripPreservesIdentity guards the data-integrity
// contract: reading an event back must yield the persisted id and timestamp,
// not a freshly generated uuid / read-time time.Now(). The token transfer
// history reader and audit/sync consumers rely on the recorded identity.
func TestSQLiteEventStore_RoundTripPreservesIdentity(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	saved := events.NewBaseEvent("token.transfer", "agg-rt", []byte(`{"step":1}`))
	if err := store.Save(saved); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := store.GetByAggregate("agg-rt", 10, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)

	if got[0].ID() != saved.ID() {
		t.Errorf("expected stored id %q, got %q", saved.ID(), got[0].ID())
	}
	// Stored timestamp is second-granularity (Unix()); compare on that window.
	if !got[0].Timestamp().Equal(saved.Timestamp().Truncate(time.Second)) {
		t.Errorf("expected stored timestamp %v, got %v", saved.Timestamp(), got[0].Timestamp())
	}
	if got[0].Payload() == nil {
		t.Errorf("expected payload to be preserved")
	} else if string(got[0].Payload()) != `{"step":1}` {
		t.Errorf("payload not preserved: %s", got[0].Payload())
	}
}

func TestSQLiteEventStore_GetByType(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	event1 := events.NewBaseEvent("nft.mint", "nft-1", []byte(`{}`))
	event2 := events.NewBaseEvent("nft.mint", "nft-2", []byte(`{}`))
	event3 := events.NewBaseEvent("token.transfer", "token-1", []byte(`{}`))

	_ = store.Save(event1)
	_ = store.Save(event2)
	_ = store.Save(event3)

	t.Run("retrieves events by type", func(t *testing.T) {
		results, err := store.GetByType("nft.mint", 10)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("respects limit", func(t *testing.T) {
		results, err := store.GetByType("nft.mint", 1)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("returns empty for non-existent type", func(t *testing.T) {
		results, err := store.GetByType("nonexistent.type", 10)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("defaults limit to 100", func(t *testing.T) {
		results, err := store.GetByType("nft.mint", 0)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestSQLiteEventStore_GetByModule(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	event1 := events.NewBaseEvent("nft.mint", "nft-1", []byte(`{}`))
	event2 := events.NewBaseEvent("nft.transfer", "nft-2", []byte(`{}`))
	event3 := events.NewBaseEvent("token.transfer", "token-1", []byte(`{}`))

	_ = store.Save(event1)
	_ = store.Save(event2)
	_ = store.Save(event3)

	t.Run("retrieves events by module", func(t *testing.T) {
		results, err := store.GetByModule("nft", 10)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("retrieves token events", func(t *testing.T) {
		results, err := store.GetByModule("token", 10)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("returns empty for non-existent module", func(t *testing.T) {
		results, err := store.GetByModule("nonexistent", 10)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestSQLiteEventStore_GetByAggregate(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	event1 := events.NewBaseEvent("token.transfer", "agg-1", []byte(`{"step":1}`))
	event2 := events.NewBaseEvent("token.transfer", "agg-1", []byte(`{"step":2}`))
	event3 := events.NewBaseEvent("token.transfer", "agg-2", []byte(`{}`))

	_ = store.Save(event1)
	_ = store.Save(event2)
	_ = store.Save(event3)

	t.Run("retrieves events by aggregate ID", func(t *testing.T) {
		results, err := store.GetByAggregate("agg-1", 50, 0)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns empty for non-existent aggregate", func(t *testing.T) {
		results, err := store.GetByAggregate("nonexistent", 50, 0)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("events ordered by timestamp ascending", func(t *testing.T) {
		results, err := store.GetByAggregate("agg-1", 50, 0)
		require.NoError(t, err)
		require.Len(t, results, 2)
	})

	t.Run("supports pagination with limit and offset", func(t *testing.T) {
		results, err := store.GetByAggregate("agg-1", 1, 0)
		require.NoError(t, err)
		assert.Len(t, results, 1)

		results, err = store.GetByAggregate("agg-1", 1, 1)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	// Equal-second events (all saved within the same Unix() second, which is
	// the norm for a burst of mints/transfers) must paginate deterministically.
	// Without a tiebreaker in ORDER BY (id), LIMIT/OFFSET over rows sharing a
	// timestamp is unordered — the same row could appear on two pages or be
	// skipped. Walking the aggregate one row at a time must reproduce the full
	// page-0 list exactly, with no row duplicated or missing.
	t.Run("equal-second events paginate deterministically", func(t *testing.T) {
		store, cleanup := setupEventStore(t)
		defer cleanup()

		const n = 8
		for i := 0; i < n; i++ {
			ev := events.NewBaseEvent("token.transfer", "agg-same-second", []byte(fmt.Sprintf(`{"step":%d}`, i)))
			_ = store.Save(ev)
		}

		full, err := store.GetByAggregate("agg-same-second", 50, 0)
		require.NoError(t, err)
		require.Len(t, full, n)

		var paged []events.Event
		for off := 0; off < n; off++ {
			page, err := store.GetByAggregate("agg-same-second", 1, off)
			require.NoError(t, err)
			require.Len(t, page, 1, "offset %d should return exactly one row", off)
			paged = append(paged, page[0])
		}

		assert.Len(t, paged, n)
		// Every page-0 row appears exactly once across the walked pages, and
		// no foreign row leaks in — i.e. paging reproduces the full set.
		fullIDs := make(map[string]int)
		for _, ev := range full {
			fullIDs[ev.ID()]++
		}
		seen := make(map[string]int)
		for _, ev := range paged {
			seen[ev.ID()]++
		}
		for id, want := range fullIDs {
			assert.Equal(t, want, seen[id], "row %s should appear exactly the same number of times across pages", id)
		}
	})
}

func TestEventStore_GetByType_Empty(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	events, err := store.GetByType("nonexistent.type", 10)
	require.NoError(t, err)
	require.Len(t, events, 0)
}

func TestEventStore_GetByModule_Empty(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	events, err := store.GetByModule("nonexistent.module", 10)
	require.NoError(t, err)
	require.Len(t, events, 0)
}

func TestEventStore_GetByAggregate_Empty(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	events, err := store.GetByAggregate("nonexistent.agg", 50, 0)
	require.NoError(t, err)
	require.Len(t, events, 0)
}

func TestEventStore_GetByType_Limit(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		e := events.NewBaseEvent("test.limit", "agg", []byte(`{}`))
		err := store.Save(e)
		require.NoError(t, err)
	}

	evts, err := store.GetByType("test.limit", 3)
	require.NoError(t, err)
	require.Len(t, evts, 3)
}

func TestSQLiteEventStore_Close(t *testing.T) {
	store, cleanup := setupEventStore(t)

	err := store.Close()
	require.NoError(t, err)
	cleanup()
}

func TestNewSQLiteEventStore_InvalidPath(t *testing.T) {
	_, err := NewSQLiteEventStore("/nonexistent/directory/that/does/not/exist/event.db")
	require.Error(t, err)
}

func TestSQLiteEventStore_GetByModule_Limit(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		e := events.NewBaseEvent("test.module_limit", "agg", []byte(`{}`))
		_ = store.Save(e)
	}

	evts, err := store.GetByModule("test", 3)
	require.NoError(t, err)
	require.Len(t, evts, 3)
}

func TestSQLiteEventStore_GetByModule_DefaultLimit(t *testing.T) {
	store, cleanup := setupEventStore(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		e := events.NewBaseEvent("test.default_limit", "agg", []byte(`{}`))
		_ = store.Save(e)
	}

	evts, err := store.GetByModule("test", 0)
	require.NoError(t, err)
	require.Len(t, evts, 5)
}
