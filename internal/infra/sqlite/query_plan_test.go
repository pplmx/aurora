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

// TestTokenListingUsesOrderedIndex covers the paged collection listing used by
// `token list`, GET /api/v1/token/list and the desktop Tokens card. Without the
// composite index SQLite materialises and sorts the whole table before the
// LIMIT can apply — measured as "USE TEMP B-TREE FOR ORDER BY" before the
// index existed.
func TestTokenListingUsesOrderedIndex(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	details := queryPlanDetails(t, repo.GetDB(),
		`SELECT id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at
		 FROM tokens ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, 100, 0)
	requireNoTempBTree(t, details, "token listing")
	require.Contains(t, strings.Join(details, " "), "idx_tokens_created",
		"the token listing should read the ordered index, got: %v", details)
}

// TestNFTListingNeedsNoTempBTree proves the NFT collection listing is ordered
// by the implicit rowid key, which SQLite walks without a sort. The test exists
// so a future change to that ORDER BY cannot silently reintroduce a sort.
func TestNFTListingNeedsNoTempBTree(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	details := queryPlanDetails(t, repo.GetDB(),
		`SELECT id, name, description, image_url, token_uri, owner, creator, block_height, timestamp
		 FROM nfts ORDER BY rowid DESC LIMIT ? OFFSET ?`, 100, 0)
	requireNoTempBTree(t, details, "nft listing")
}

// TestCollectionListsUseTheirIndexes covers the collection reads the operator
// sees as lists. Oracle sources and voters are newest-first full reads, while
// the voting candidate/session lists page with a deterministic id tie-breaker.
// Without the matching indexes every row is materialised and sorted before
// LIMIT can apply (measured: "SCAN ... USE TEMP B-TREE FOR ORDER BY").
func TestCollectionListsUseTheirIndexes(t *testing.T) {
	oracleRepo, oracleCleanup := setupOracleTestDB(t)
	defer oracleCleanup()
	requireNoTempBTree(t, queryPlanDetails(t, oracleRepo.GetDB(),
		`SELECT id, name, url, type, method, headers, path, interval, enabled, created_at
		 FROM data_sources ORDER BY created_at DESC`), "oracle source list")

	votingRepo, votingCleanup := setupVotingTestDB(t)
	defer votingCleanup()
	requireNoTempBTree(t, queryPlanDetails(t, votingRepo.db,
		`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters ORDER BY registered_at DESC`),
		"voter list")
	candidatePlan := queryPlanDetails(t, votingRepo.db,
		`SELECT id, name, party, program, description, image_url, vote_count, created_at
		 FROM candidates ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, 100, 0)
	requireNoTempBTree(t, candidatePlan, "candidate list")
	require.Contains(t, strings.Join(candidatePlan, " "), "idx_candidates_page",
		"the candidate page should read its ordered index")

	sessionPlan := queryPlanDetails(t, votingRepo.db,
		`SELECT id, title, description, start_time, end_time, status, candidates, created_at
		 FROM voting_sessions ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, 100, 0)
	requireNoTempBTree(t, sessionPlan, "session list")
	require.Contains(t, strings.Join(sessionPlan, " "), "idx_sessions_page",
		"the session page should read its ordered index")
}

// TestLotteryHistoryUsesRecencyIndex covers the deterministic newest-first
// order: timestamp DESC, block_height DESC, id. Ordering by timestamp alone
// used the single-column index and could not break same-second ties; adding
// the tiebreakers to that plan made SQLite sort in a temp B-tree (measured as
// "USE TEMP B-TREE FOR LAST TERM OF ORDER BY"). The composite index keeps the
// order a direct walk, and the repository bootstrap drops the now-redundant
// single-column index.
func TestLotteryHistoryUsesRecencyIndex(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	details := queryPlanDetails(t, repo.GetDB(),
		`SELECT l.id, l.block_height, l.seed, l.participants, l.winners, l.winner_addresses,
		        l.vrf_proof, l.vrf_output, COALESCE(k.public_key, ''), l.timestamp, l.verified
		 FROM lottery_records l
		 LEFT JOIN lottery_vrf_keys k ON k.record_id = l.id
		 `+lotteryNewestFirst+`
		 LIMIT ? OFFSET ?`, 100, 0)
	requireNoTempBTree(t, details, "lottery history")
	require.Contains(t, strings.Join(details, " "), "idx_lottery_recency",
		"lottery history should read the recency index, got: %v", details)

	var oldIndex int
	require.NoError(t, repo.GetDB().QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_lottery_timestamp'`,
	).Scan(&oldIndex))
	require.Zero(t, oldIndex, "the timestamp-only index is subsumed by idx_lottery_recency")
}
