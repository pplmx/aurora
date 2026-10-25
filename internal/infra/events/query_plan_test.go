package events

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// queryPlanDetails returns the EXPLAIN QUERY PLAN detail lines for a statement.
func queryPlanDetails(t *testing.T, db *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var details []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notUsed, &detail))
		details = append(details, detail)
	}
	require.NoError(t, rows.Err())
	return details
}

func requireNoTempBTree(t *testing.T, details []string, what string) {
	t.Helper()
	require.NotEmpty(t, details, "%s: expected at least one plan row", what)
	for _, d := range details {
		require.NotContains(t, strings.ToUpper(d), "TEMP B-TREE",
			"%s: plan still sorts in a temp B-tree: %s", what, d)
	}
}

// TestEventHistoryUsesOrderedIndex covers the reads behind `token history`,
// `nft history` and the per-account listings. All of them filter on the
// aggregate and order by timestamp; before idx_events_agg_ts SQLite searched
// idx_events_agg and then sorted every matching row in a temp B-tree, so the
// LIMIT could not stop early (measured with EXPLAIN QUERY PLAN).
func TestEventHistoryUsesOrderedIndex(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	details := queryPlanDetails(t, store.db,
		`SELECT id, event_type, module, agg_id, payload, timestamp
		 FROM events WHERE agg_id = ? ORDER BY timestamp DESC, id DESC LIMIT ?`,
		"agg-1", 50)
	requireNoTempBTree(t, details, "event history by aggregate")

	details = queryPlanDetails(t, store.db,
		`SELECT id, event_type, module, agg_id, payload, timestamp
		 FROM events WHERE agg_id = ? AND event_type = ?
		 ORDER BY timestamp ASC, id ASC LIMIT ? OFFSET ?`,
		"agg-1", "token.transfer", 50, 0)
	requireNoTempBTree(t, details, "event history by aggregate and type")

	details = queryPlanDetails(t, store.db,
		`SELECT id, event_type, module, agg_id, payload, timestamp
		 FROM events WHERE agg_id = ? AND event_type = ?
		   AND (json_extract(CAST(payload AS TEXT), ?) = ?
		        OR json_extract(CAST(payload AS TEXT), ?) = ?)
		 ORDER BY timestamp ASC, id ASC LIMIT ? OFFSET ?`,
		"agg-1", "token.transfer", "$.from", "a", "$.to", "a", 50, 0)
	requireNoTempBTree(t, details, "token transfer history (Either predicate)")
}
