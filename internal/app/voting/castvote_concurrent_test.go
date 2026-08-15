package voting

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCastVoteTestDB opens a real file-backed SQLite database with the
// voting schema and WAL mode (matching production: the shared aurora.db is
// switched to WAL by the token repository's createTables). A file DB gives
// real multi-connection concurrency, unlike the :memory: single-connection
// setup used inside the sqlite package's own tests.
func setupCastVoteTestDB(t *testing.T) (*sqlite.VotingRepository, *sqlite.TxManager, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "castvote.db")
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
	require.NoError(t, err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS votes (
			id TEXT PRIMARY KEY,
			voter_pk TEXT NOT NULL,
			candidate_id TEXT NOT NULL,
			signature TEXT,
			message TEXT,
			timestamp INTEGER,
			block_height INTEGER
		);
		CREATE TABLE IF NOT EXISTS voters (
			public_key TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			has_voted INTEGER DEFAULT 0,
			vote_hash TEXT,
			registered_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS candidates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			party TEXT,
			program TEXT,
			description TEXT,
			image_url TEXT,
			vote_count INTEGER DEFAULT 0,
			created_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS voting_sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			start_time INTEGER,
			end_time INTEGER,
			status TEXT,
			candidates TEXT,
			created_at INTEGER
		);
	`)
	require.NoError(t, err)

	repo := sqlite.NewVotingRepository(db)
	txMgr := sqlite.NewTxManager(db)
	cleanup := func() { _ = db.Close() }
	return repo, txMgr, cleanup
}

// TestCastVoteUseCase_ConcurrentDistinctVoters proves that N concurrent
// ballots for the SAME candidate through the real transactional path lose no
// tally increments and leave a consistent votes<->voters state. This is the
// lost-update regression (decision-atomic-primitives-over-rmw) exercised end
// to end: claim + vote row + tally increment commit as one transaction per
// ballot.
func TestCastVoteUseCase_ConcurrentDistinctVoters(t *testing.T) {
	repo, txMgr, cleanup := setupCastVoteTestDB(t)
	defer cleanup()

	service := voting.NewEd25519Service()

	candidate := voting.NewCandidate("Alice", "Party A", "Platform")
	require.NoError(t, repo.SaveCandidate(candidate))

	now := time.Now().Unix()
	session := voting.NewSession("Concurrent Election", "", []string{candidate.ID}, now-3600, now+3600)
	require.NoError(t, repo.SaveSession(session))

	const voters = 12
	privKeys := make([]string, voters)
	for i := 0; i < voters; i++ {
		pub, priv, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)
		pk := base64.StdEncoding.EncodeToString(pub)
		privKeys[i] = base64.StdEncoding.EncodeToString(priv)
		require.NoError(t, repo.SaveVoter(&voting.Voter{
			PublicKey:    pk,
			Name:         fmt.Sprintf("voter-%d", i),
			RegisteredAt: now,
		}))
	}

	var wg sync.WaitGroup
	errs := make([]error, voters)
	start := make(chan struct{})
	for i := 0; i < voters; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			uc := NewCastVoteUseCase(repo, service, txMgr)
			_, err := uc.Execute(CastVoteRequest{
				VoterPublicKey: base64.StdEncoding.EncodeToString(mustPubFromPriv(t, privKeys[idx])),
				CandidateID:    candidate.ID,
				PrivateKey:     privKeys[idx],
				SessionID:      session.ID,
			})
			errs[idx] = err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "voter %d ballot failed", i)
	}

	got, err := repo.GetCandidate(candidate.ID)
	require.NoError(t, err)
	assert.Equal(t, voters, got.VoteCount,
		"no tally increment may be lost under concurrency (lost-update regression)")

	// votes<->voters invariant: every voter is marked, every vote row exists.
	voteRows := 0
	for i := 0; i < voters; i++ {
		pk := base64.StdEncoding.EncodeToString(mustPubFromPriv(t, privKeys[i]))
		v, err := repo.GetVoter(pk)
		require.NoError(t, err)
		assert.True(t, v.HasVoted, "voter %d must be marked", i)
		rows, err := repo.GetVotesByVoter(pk)
		require.NoError(t, err)
		voteRows += len(rows)
	}
	assert.Equal(t, voters, voteRows, "exactly one vote row per voter")
}

// TestCastVoteUseCase_ConcurrentSameVoterOnlyOneWins proves that concurrent
// ballots from the SAME voter result in exactly one accepted vote, through
// the real transactional path (TryMarkVoted's conditional UPDATE inside the
// transaction; all losers abort with ErrAlreadyVoted and no side effects).
func TestCastVoteUseCase_ConcurrentSameVoterOnlyOneWins(t *testing.T) {
	repo, txMgr, cleanup := setupCastVoteTestDB(t)
	defer cleanup()

	service := voting.NewEd25519Service()

	candidate := voting.NewCandidate("Bob", "Party B", "Platform")
	require.NoError(t, repo.SaveCandidate(candidate))

	now := time.Now().Unix()
	session := voting.NewSession("Race Election", "", []string{candidate.ID}, now-3600, now+3600)
	require.NoError(t, repo.SaveSession(session))

	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	pk := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)
	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: pk, Name: "racer", RegisteredAt: now}))

	const attempts = 8
	var wg sync.WaitGroup
	var wins int32
	var already int32
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			uc := NewCastVoteUseCase(repo, service, txMgr)
			_, err := uc.Execute(CastVoteRequest{
				VoterPublicKey: pk,
				CandidateID:    candidate.ID,
				PrivateKey:     privB64,
				SessionID:      session.ID,
			})
			switch {
			case err == nil:
				atomic.AddInt32(&wins, 1)
			case errors.Is(err, voting.ErrAlreadyVoted):
				atomic.AddInt32(&already, 1)
			default:
				t.Errorf("unexpected ballot error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), wins, "exactly one concurrent ballot may win")
	assert.Equal(t, int32(attempts-1), already, "all other ballots must be rejected as already voted")

	got, err := repo.GetCandidate(candidate.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.VoteCount, "tally must count the single winning ballot exactly once")

	rows, err := repo.GetVotesByVoter(pk)
	require.NoError(t, err)
	assert.Len(t, rows, 1, "exactly one vote row for the racing voter")
}

func mustPubFromPriv(t *testing.T, privB64 string) ed25519.PublicKey {
	t.Helper()
	priv, err := base64.StdEncoding.DecodeString(privB64)
	require.NoError(t, err)
	return ed25519.PrivateKey(priv).Public().(ed25519.PublicKey)
}
