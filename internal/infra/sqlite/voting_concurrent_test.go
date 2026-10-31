package sqlite

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVotingRepository_SaveVote_ConcurrentNoDoubleVote proves the
// (session_id, voter_pk) claim enforces one ballot per voter in a session
// under concurrent saves.
func TestVotingRepository_SaveVote_ConcurrentNoDoubleVote(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	// :memory: SQLite databases are per-connection. With a real pool,
	// concurrent goroutines would each see their own DB. Serialize
	// through one connection so the test exercises the conditional
	// UPDATE atomically against a single, shared database.
	repo.db.SetMaxOpenConns(1)

	voter := &voting.Voter{
		PublicKey:    "test-pk",
		Name:         "Alice",
		HasVoted:     false,
		RegisteredAt: 1,
	}
	require.NoError(t, repo.SaveVoter(voter))

	const goroutines = 16
	var wg sync.WaitGroup
	var successCount int32
	var alreadyVotedCount int32
	var notFoundCount int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.SaveVote(voting.NewSessionVote(
				"test-pk", "candidate-1", "session-1", "signature", "message",
			))
			switch {
			case err == nil:
				atomic.AddInt32(&successCount, 1)
			case errors.Is(err, ErrAlreadyVoted):
				atomic.AddInt32(&alreadyVotedCount, 1)
			case errors.Is(err, ErrNotFound):
				atomic.AddInt32(&notFoundCount, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), successCount, "exactly one goroutine should succeed in marking the voter")
	assert.Equal(t, int32(goroutines-1), alreadyVotedCount, "all others should be rejected as already-voted")
	assert.Equal(t, int32(0), notFoundCount)

	got, err := repo.GetVoter("test-pk")
	require.NoError(t, err)
	assert.True(t, got.HasVoted, "voter must be marked as voted")

	var claims, votes int
	require.NoError(t, repo.db.QueryRow(
		`SELECT COUNT(*) FROM session_vote_claims
		 WHERE session_id = 'session-1' AND voter_pk = 'test-pk'`,
	).Scan(&claims))
	require.NoError(t, repo.db.QueryRow(
		`SELECT COUNT(*) FROM votes WHERE voter_pk = 'test-pk'`,
	).Scan(&votes))
	assert.Equal(t, 1, claims)
	assert.Equal(t, 1, votes, "losing claims must roll back their vote rows")
}

func TestVotingRepository_SaveVote_SessionClaimNotFound(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	err := repo.SaveVote(voting.NewSessionVote(
		"nonexistent", "candidate-1", "session-1", "signature", "message",
	))
	assert.ErrorIs(t, err, ErrNotFound)

	var votes int
	require.NoError(t, repo.db.QueryRow(`SELECT COUNT(*) FROM votes`).Scan(&votes))
	assert.Zero(t, votes, "a missing voter must roll back the vote row")
}

func TestVotingRepository_SaveVote_ClaimIsScopedToSession(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	voter := &voting.Voter{
		PublicKey:    "test-pk-2",
		Name:         "Bob",
		HasVoted:     false,
		RegisteredAt: 1,
	}
	require.NoError(t, repo.SaveVoter(voter))

	require.NoError(t, repo.SaveVote(voting.NewSessionVote(
		"test-pk-2", "candidate-1", "session-1", "first-signature", "first-message",
	)))
	err := repo.SaveVote(voting.NewSessionVote(
		"test-pk-2", "candidate-1", "session-1", "second-signature", "second-message",
	))
	assert.ErrorIs(t, err, ErrAlreadyVoted)

	got, err := repo.GetVoter("test-pk-2")
	require.NoError(t, err)
	assert.True(t, got.HasVoted)
	assert.Equal(t, "first-message", got.VoteHash, "the lifetime reference keeps the first ballot")

	require.NoError(t, repo.SaveVote(voting.NewSessionVote(
		"test-pk-2", "candidate-1", "session-2", "third-signature", "third-message",
	)), "the same voter must be eligible in another session")

	var claims int
	require.NoError(t, repo.db.QueryRow(
		`SELECT COUNT(*) FROM session_vote_claims WHERE voter_pk = 'test-pk-2'`,
	).Scan(&claims))
	assert.Equal(t, 2, claims)
}

// TestVotingRepository_SaveVoter_PreservesHasVotedOnConflict is a
// regression test for the INSERT OR REPLACE footgun in SaveVoter.
//
// Pre-fix behaviour: SaveVoter used INSERT OR REPLACE which, on
// public_key conflict, DELETED the existing row and reinserted
// with the caller's fields. A caller passing has_voted=false on
// an existing public_key would silently wipe the recorded vote
// and undo the atomic guarantee from SaveVote's session claim.
//
// Post-fix behaviour: SaveVoter uses ON CONFLICT DO UPDATE that
// only touches `name` and `registered_at`. has_voted and
// vote_hash are preserved on conflict.
func TestVotingRepository_SaveVoter_PreservesHasVotedOnConflict(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	pk := "test-voter-pk"
	original := &voting.Voter{
		PublicKey:    pk,
		Name:         "Original Name",
		HasVoted:     false,
		RegisteredAt: 1000,
	}
	require.NoError(t, repo.SaveVoter(original))

	// Mark the voter as having cast a ballot through the session claim.
	require.NoError(t, repo.SaveVote(voting.NewSessionVote(
		pk, "candidate-1", "session-1", "signature", "vote-hash-abc",
	)))

	// Now a caller tries to "re-register" the same public_key
	// with has_voted=false and a different name. Pre-fix this
	// would silently undo the vote. Post-fix the vote is
	// preserved.
	reregister := &voting.Voter{
		PublicKey:    pk,
		Name:         "Renamed",
		HasVoted:     false,
		VoteHash:     "",
		RegisteredAt: 2000,
	}
	require.NoError(t, repo.SaveVoter(reregister))

	got, err := repo.GetVoter(pk)
	require.NoError(t, err)
	require.NotNil(t, got)

	if !got.HasVoted {
		t.Errorf("SaveVoter conflict wiped has_voted: expected true, got false (Round 14 atomic guarantee violated)")
	}
	if got.VoteHash != "vote-hash-abc" {
		t.Errorf("SaveVoter conflict wiped vote_hash: expected %q, got %q", "vote-hash-abc", got.VoteHash)
	}
	if got.Name != "Renamed" {
		t.Errorf("SaveVoter conflict failed to update name: expected %q, got %q", "Renamed", got.Name)
	}
	if got.RegisteredAt != 2000 {
		t.Errorf("SaveVoter conflict failed to update registered_at: expected 2000, got %d", got.RegisteredAt)
	}
}

// TestVotingRepository_IncrementCandidateVoteCount_ConcurrentNoLostUpdate
// proves concurrent votes to the same candidate never lose an increment.
//
// The previous CastVoteUseCase flow did read candidate.VoteCount, increment
// in memory, then write the whole row back via UpdateCandidate. Two voters
// voting for the same candidate concurrently both read N, both computed N+1,
// and both wrote N+1 — one vote silently uncounted. The conditional UPDATE
// (vote_count = vote_count + 1) serializes writers so every call lands.
func TestVotingRepository_IncrementCandidateVoteCount_ConcurrentNoLostUpdate(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	// Same single-connection note as the SaveVote claim test above.
	repo.db.SetMaxOpenConns(1)

	candidate := &voting.Candidate{
		ID:        "cand-1",
		Name:      "Alice",
		VoteCount: 0,
	}
	require.NoError(t, repo.SaveCandidate(candidate))

	const goroutines = 16
	var wg sync.WaitGroup
	var errCount int32
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := repo.IncrementCandidateVoteCount("cand-1"); err != nil {
				atomic.AddInt32(&errCount, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(0), errCount, "no increment should fail")
	got, err := repo.GetCandidate("cand-1")
	require.NoError(t, err)
	assert.Equal(t, goroutines, got.VoteCount, "every concurrent vote must be counted exactly once")
}

// TestVotingRepository_IncrementCandidateVoteCount_NotFound proves the
// increment maps a missing candidate to the not-found sentinel (relevant when
// a candidate is deleted between the use case's existence check and the
// atomic increment).
func TestVotingRepository_IncrementCandidateVoteCount_NotFound(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	err := repo.IncrementCandidateVoteCount("missing-candidate")
	assert.ErrorIs(t, err, ErrNotFound)
}
