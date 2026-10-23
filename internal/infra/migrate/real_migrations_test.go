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
// bug: migrations/000001 began with `PRAGMA journal_mode=WAL;`, which
// golang-migrate's sqlite3 driver executes inside an explicit transaction
// where WAL mode cannot be switched → every real migration run failed with
// "cannot change into wal mode from within a transaction". The migrate
// package's own tests used synthetic migrations, so the real files were
// never exercised.
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
	defer db.Close()

	for _, table := range []string{
		"blocks", "lottery_records",
		"votes", "vote_sessions", "voters", "candidates", "voting_sessions",
		"nfts", "nft_operations",
		"tokens", "accounts", "allowances",
		"data_sources", "oracle_data",
	} {
		assertTableExists(t, db, table)
	}
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
