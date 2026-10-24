package sqlite

import (
	"database/sql"
	"path/filepath"
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

// requireNoTempBTree fails when the plan sorts in a temporary B-tree. Oracle
// history and NFT operation history are read per parent row in descending
// timestamp order; without the composite index SQLite has to materialize and
// sort every matching row before LIMIT can apply, so the LIMIT stops being an
// early exit (v1.100).
func requireNoTempBTree(t *testing.T, details []string, what string) {
	t.Helper()
	require.NotEmpty(t, details, "%s: expected at least one plan row", what)
	for _, d := range details {
		require.NotContains(t, strings.ToUpper(d), "TEMP B-TREE",
			"%s: plan still sorts in a temp B-tree: %s", what, d)
	}
}

func TestOracleHistoryUsesOrderedIndex(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewOracleRepository(filepath.Join(tmpDir, "oracle_plan.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	details := queryPlanDetails(t, repo.GetDB(),
		`SELECT id, source_id, value, raw_response, timestamp, block_height
		 FROM oracle_data WHERE source_id = ? ORDER BY timestamp DESC LIMIT ?`,
		"source-1", 100)
	requireNoTempBTree(t, details, "oracle history")
	require.Contains(t, strings.Join(details, " "), "idx_oracle_data_source_ts",
		"oracle history should read the composite ordered index, got: %v", details)
}

func TestNFTOperationHistoryUsesOrderedIndex(t *testing.T) {
	tmpDir := t.TempDir()
	repo, err := NewNFTRepository(filepath.Join(tmpDir, "nft_plan.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	details := queryPlanDetails(t, repo.GetDB(),
		`SELECT id, nft_id, type, from_addr, to_addr, signature, block_height, timestamp
		 FROM nft_operations WHERE nft_id = ? ORDER BY timestamp DESC LIMIT ?`,
		"nft-1", maxNFTHistoryOps)
	requireNoTempBTree(t, details, "nft operation history")
	require.Contains(t, strings.Join(details, " "), "idx_nft_ops_nft_ts",
		"nft operation history should read the composite ordered index, got: %v", details)
}
