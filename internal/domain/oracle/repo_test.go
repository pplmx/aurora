package oracle

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInmemRepo_ReadsMirrorTheStoredOrder pins the read contract the SQLite
// repository implements, because handler and use-case tests read through this
// in-memory store: `timestamp DESC, rowid ASC`. A Go map has no order, so the
// first draft returned readings oldest first and picked the earlier of two
// same-second readings as "latest" — the opposite of what `/oracle/query` and
// `/oracle/latest` answer in production.
func TestInmemRepo_ReadsMirrorTheStoredOrder(t *testing.T) {
	repo := NewInmemRepo()
	// Inserted oldest first, so a passing test cannot be explained by
	// reproducible map iteration.
	require.NoError(t, repo.SaveData(&OracleData{ID: "old", SourceID: "s", Timestamp: 100}))
	require.NoError(t, repo.SaveData(&OracleData{ID: "mid", SourceID: "s", Timestamp: 200}))
	require.NoError(t, repo.SaveData(&OracleData{ID: "new", SourceID: "s", Timestamp: 300}))

	all, err := repo.GetDataBySource("s", 0)
	require.NoError(t, err)
	require.Equal(t, []string{"new", "mid", "old"}, dataIDs(all),
		"the query endpoint lists newest first")

	page, err := repo.GetDataBySource("s", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"new", "mid"}, dataIDs(page),
		"a bounded page is the head of the same order, not an arbitrary subset")

	latest, err := repo.GetLatestData("s")
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.Equal(t, "new", latest.ID)

	other, err := repo.GetDataBySource("missing", 0)
	require.NoError(t, err)
	require.Empty(t, other)
	nothing, err := repo.GetLatestData("missing")
	require.NoError(t, err)
	require.Nil(t, nothing, "an empty source is an empty read, not an error")
}

// TestInmemRepo_SameSecondReadingsKeepInsertionOrder covers the tie the store
// breaks with rowid: two readings in the same second must not swap between
// reads, and the first one written is the one /oracle/latest reports.
func TestInmemRepo_SameSecondReadingsKeepInsertionOrder(t *testing.T) {
	repo := NewInmemRepo()
	require.NoError(t, repo.SaveData(&OracleData{ID: "first", SourceID: "s", Timestamp: 500}))
	require.NoError(t, repo.SaveData(&OracleData{ID: "second", SourceID: "s", Timestamp: 500}))

	for i := 0; i < 5; i++ {
		all, err := repo.GetDataBySource("s", 0)
		require.NoError(t, err)
		require.Equal(t, []string{"first", "second"}, dataIDs(all),
			"same-second readings keep the order they were written in")
	}

	latest, err := repo.GetLatestData("s")
	require.NoError(t, err)
	require.Equal(t, "first", latest.ID, "latest is the first row of the same list")
}

// TestInmemRepo_RewritingAReadingMovesItToTheEndOfItsSecond mirrors the store's
// INSERT OR REPLACE: the rewrite takes a fresh rowid, so the same second lists
// the corrected row last instead of where the original landed.
func TestInmemRepo_RewritingAReadingMovesItToTheEndOfItsSecond(t *testing.T) {
	repo := NewInmemRepo()
	require.NoError(t, repo.SaveData(&OracleData{ID: "first", SourceID: "s", Timestamp: 500, Value: "1"}))
	require.NoError(t, repo.SaveData(&OracleData{ID: "second", SourceID: "s", Timestamp: 500}))
	require.NoError(t, repo.SaveData(&OracleData{ID: "first", SourceID: "s", Timestamp: 500, Value: "2"}))

	all, err := repo.GetDataBySource("s", 0)
	require.NoError(t, err)
	require.Equal(t, []string{"second", "first"}, dataIDs(all),
		"a rewritten row takes the newest position within its second, as INSERT OR REPLACE does")
	require.Equal(t, "2", all[1].Value, "the corrected value is what the row now holds")
}

func dataIDs(rows []*OracleData) []string {
	out := make([]string, len(rows))
	for i, d := range rows {
		out[i] = d.ID
	}
	return out
}
