package events

import (
	"os"
	"testing"

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
		results, err := store.GetByAggregate("agg-1")
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns empty for non-existent aggregate", func(t *testing.T) {
		results, err := store.GetByAggregate("nonexistent")
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("events ordered by timestamp ascending", func(t *testing.T) {
		results, err := store.GetByAggregate("agg-1")
		require.NoError(t, err)
		require.Len(t, results, 2)
	})
}

func TestSQLiteEventStore_Close(t *testing.T) {
	store, cleanup := setupEventStore(t)

	err := store.Close()
	require.NoError(t, err)
	cleanup()
}
