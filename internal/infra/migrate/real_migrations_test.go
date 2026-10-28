package migrate

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// realMigrationsDir resolves the repository's migrations/ directory from
// this source file's location (tests must not depend on the process cwd).
func realMigrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations"))
}

// TestRealMigrations_Apply is the regression test for the latent migration
// bug: migrations/000001 began with `PRAGMA journal_mode=WAL;`, which the
// migration runner executes inside an explicit transaction where WAL mode
// cannot be switched. The migrate package's own tests used synthetic
// migrations, so the real files were never exercised.
//
// This test applies the ACTUAL checkout migrations and asserts the tables
// they promise exist. It must stay green as long as the migrations apply.
func TestRealMigrations_Apply(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err, "New with real migrations dir")
	defer func() { _ = m.Close() }()

	// Up(0) applies every pending migration (steps<=0 means "all"); Up(N)
	// with N larger than the number of migrations errors with "limit short".
	version, err := m.Up(0)
	require.NoError(t, err, "applying real migrations must not fail")
	require.NotZero(t, version, "expected at least one migration applied")

	t.Logf("migrations applied to version %d", version)

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	for _, table := range []string{
		"blocks", "lottery_records", "lottery_vrf_keys",
		"votes", "vote_sessions", "session_vote_claims",
		"voters", "candidates", "voting_sessions",
		"nfts", "nft_operations",
		"tokens", "accounts", "allowances",
		"data_sources", "oracle_data",
	} {
		assertTableExists(t, db, table)
	}

	// Migration 000007 replaces the timestamp-only lottery index with the
	// composite recency index the repository now orders by.
	var oldTimestampIndex, recencyIndex int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_lottery_timestamp'`,
	).Scan(&oldTimestampIndex))
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_lottery_recency'`,
	).Scan(&recencyIndex))
	assert.Zero(t, oldTimestampIndex, "the timestamp-only index is subsumed by idx_lottery_recency")
	assert.Equal(t, 1, recencyIndex, "migration 000007 must create the recency index")
}

// TestRealMigrations_LotteryVRFKeyUpgrade proves migration 000004 adds the
// idempotent proof-key side table to an existing version-3 database without
// touching already stored lottery records.
func TestRealMigrations_LotteryVRFKeyUpgrade(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	_, err = m.Up(3)
	require.NoError(t, err)
	require.NoError(t, m.Close())

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO lottery_records
			(id, block_height, seed, participants, winners, winner_addresses,
			 vrf_proof, vrf_output, timestamp, verified)
		VALUES ('legacy', 1, 'legacy-seed', '["alice"]', '["alice"]', '["addr"]',
			'proof', 'output', 1, 0);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	m, err = New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	defer func() { _ = m.Close() }()
	_, err = m.Up(0)
	require.NoError(t, err, "migration 000004 must upgrade a populated version-3 database")

	db, err = sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var legacyCount int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM lottery_records WHERE id = 'legacy'`,
	).Scan(&legacyCount))
	assert.Equal(t, 1, legacyCount, "adding the key table must preserve existing draws")

	var keyCount int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM lottery_vrf_keys WHERE record_id = 'legacy'`,
	).Scan(&keyCount))
	assert.Zero(t, keyCount, "legacy draws have no recoverable public key")
}

// TestRealMigrations_UpTwice_NoChange hammers the idempotency contract:
// running Up a second time must be a no-op (ErrNoChange), not an error.
func TestRealMigrations_UpTwice_NoChange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	_, err = m.Up(0)
	require.NoError(t, err)

	_, err = m.Up(0)
	require.NoError(t, err, "second Up must be a no-op, not an error")
}

// TestRealMigrations_VotesSessionID proves migration 000003 adds the session
// mapping without rebuilding votes or losing legacy ballots. Legacy rows stay
// unattributed, while new vote/session pairs can be counted per election
// (ISS-299).
func TestRealMigrations_VotesSessionID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE votes (
			id TEXT PRIMARY KEY,
			voter_pk TEXT NOT NULL,
			candidate_id TEXT NOT NULL,
			signature TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			block_height INTEGER NOT NULL
		);
		INSERT INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		VALUES ('legacy', 'voter', 'cand', 'sig', 'msg', 1, 1);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	defer func() { _ = m.Close() }()
	_, err = m.Up(0)
	require.NoError(t, err, "upgrading a legacy votes table must not fail")

	db, err = sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var legacyCount int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM votes WHERE id = 'legacy'`).Scan(&legacyCount))
	require.Equal(t, 1, legacyCount, "migration must preserve legacy ballots")

	var legacyMappings int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM vote_sessions WHERE vote_id = 'legacy'`).Scan(&legacyMappings))
	require.Equal(t, 0, legacyMappings, "legacy ballots cannot be attributed to a session")

	_, err = db.Exec(`INSERT INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		VALUES ('new', 'voter2', 'cand', 'sig', 'msg', 2, 2)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO vote_sessions (vote_id, session_id) VALUES ('new', 's1')`)
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRow(`
		SELECT count(*)
		FROM votes v
		JOIN vote_sessions vs ON vs.vote_id = v.id
		WHERE vs.session_id = 's1' AND v.candidate_id = 'cand'
	`).Scan(&n))
	require.Equal(t, 1, n)
}

// TestRealMigrations_PreservesBootstrappedVoteSessions is the regression for
// applying 000003 after VotingRepository has already self-bootstrapped the
// schema and recorded a session vote. The migration must not rebuild votes or
// discard the mapping the repository wrote (ISS-299).
func TestRealMigrations_PreservesBootstrappedVoteSessions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	db, err := sql.Open("sqlite", sqlite.DSN(dbPath))
	require.NoError(t, err)
	repo, err := sqlite.NewVotingRepository(db)
	require.NoError(t, err)

	require.NoError(t, repo.SaveVoter(&voting.Voter{
		PublicKey: "voter", Name: "Voter", RegisteredAt: 1,
	}))
	require.NoError(t, repo.SaveVote(&voting.Vote{
		ID:             "bootstrapped",
		VoterPublicKey: "voter",
		CandidateID:    "cand",
		SessionID:      "s1",
		Signature:      "sig",
		Message:        "msg",
		Timestamp:      1,
		BlockHeight:    1,
	}))
	require.NoError(t, db.Close())

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	defer func() { _ = m.Close() }()
	_, err = m.Up(0)
	require.NoError(t, err, "migration must accept an already-bootstrapped schema")

	db, err = sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var sessionID string
	require.NoError(t, db.QueryRow(
		`SELECT session_id FROM vote_sessions WHERE vote_id = 'bootstrapped'`,
	).Scan(&sessionID))
	require.Equal(t, "s1", sessionID, "migration must preserve the repository's session mapping")

	var claimVoteID string
	require.NoError(t, db.QueryRow(
		`SELECT vote_id FROM session_vote_claims
		 WHERE session_id = 's1' AND voter_pk = 'voter'`,
	).Scan(&claimVoteID))
	require.Equal(t, "bootstrapped", claimVoteID,
		"migration must preserve the claim written by the bootstrapped repository")
}

// TestRealMigrations_BackfillsSessionVoteClaims proves the upgrade converts
// existing session ballots into per-session claims without deleting any audit
// rows. If an old database already contains two ballots from one voter in the
// same session, the earliest ballot owns the claim and all vote rows remain.
func TestRealMigrations_BackfillsSessionVoteClaims(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")

	m, err := New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	_, err = m.Up(7)
	require.NoError(t, err)
	require.NoError(t, m.Close())

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO voters (public_key, name, has_voted, vote_hash, registered_at)
		VALUES ('voter', 'Alice', 0, NULL, 1);

		INSERT INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		VALUES
			('first', 'voter', 'cand', 'sig-1', 'msg-1', 1, 1),
			('second', 'voter', 'cand', 'sig-2', 'msg-2', 2, 2),
			('duplicate', 'voter', 'cand', 'sig-3', 'msg-3', 3, 3);

		INSERT INTO vote_sessions (vote_id, session_id)
		VALUES ('first', 's1'), ('second', 's2'), ('duplicate', 's1');
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	m, err = New(dbPath, realMigrationsDir())
	require.NoError(t, err)
	defer func() { _ = m.Close() }()
	_, err = m.Up(0)
	require.NoError(t, err, "migration 000008 must upgrade existing session ballots")

	db, err = sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`
		SELECT session_id, vote_id
		FROM session_vote_claims
		WHERE voter_pk = 'voter'
		ORDER BY session_id
	`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var claims [][2]string
	for rows.Next() {
		var sessionID, voteID string
		require.NoError(t, rows.Scan(&sessionID, &voteID))
		claims = append(claims, [2]string{sessionID, voteID})
	}
	require.NoError(t, rows.Err())
	require.Equal(t, [][2]string{{"s1", "first"}, {"s2", "second"}}, claims,
		"the earliest ballot wins a duplicate historical claim")

	var votes, hasVoted int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM votes`).Scan(&votes))
	require.NoError(t, db.QueryRow(
		`SELECT has_voted FROM voters WHERE public_key = 'voter'`,
	).Scan(&hasVoted))
	assert.Equal(t, 3, votes, "migration must preserve every historical ballot")
	assert.Equal(t, 1, hasVoted, "the lifetime voter flag must reflect the backfilled claims")
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var n int
	err := db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table,
	).Scan(&n)
	require.NoError(t, err, "query sqlite_master for %s", table)
	assert.Equal(t, 1, n, "migration must create table %s", table)
}
