package sqlite

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVotingRepository_TryMarkVoted_ConcurrentNoDoubleVote proves the
// repo enforces single-vote-per-voter under concurrent CastVote calls.
//
// The TOCTOU window in the previous CastVoteUseCase implementation
// (read has_voted, decide, then UPDATE has_voted=true) let two
// concurrent requests both succeed, allowing the same voter to vote
// twice. The fix is a conditional UPDATE in TryMarkVoted that uses
// SQLite's per-connection write serialization so exactly one caller
// sees RowsAffected()==1.
func TestVotingRepository_TryMarkVoted_ConcurrentNoDoubleVote(t *testing.T) {
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
			err := repo.TryMarkVoted("test-pk", "vote-hash")
			switch err {
			case nil:
				atomic.AddInt32(&successCount, 1)
			case ErrAlreadyVoted:
				atomic.AddInt32(&alreadyVotedCount, 1)
			case ErrNotFound:
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
}

func TestVotingRepository_TryMarkVoted_NotFound(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	err := repo.TryMarkVoted("nonexistent", "vote-hash")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestVotingRepository_TryMarkVoted_Sequential(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	voter := &voting.Voter{
		PublicKey:    "test-pk-2",
		Name:         "Bob",
		HasVoted:     false,
		RegisteredAt: 1,
	}
	require.NoError(t, repo.SaveVoter(voter))

	require.NoError(t, repo.TryMarkVoted("test-pk-2", "first-hash"))
	err := repo.TryMarkVoted("test-pk-2", "second-hash")
	assert.ErrorIs(t, err, ErrAlreadyVoted)

	got, err := repo.GetVoter("test-pk-2")
	require.NoError(t, err)
	assert.True(t, got.HasVoted)
	assert.Equal(t, "first-hash", got.VoteHash, "first hash wins; second attempt must not overwrite")
}

// TestVotingRepository_SaveVoter_PreservesHasVotedOnConflict is a
// regression test for the INSERT OR REPLACE footgun in SaveVoter.
//
// Pre-fix behaviour: SaveVoter used INSERT OR REPLACE which, on
// public_key conflict, DELETED the existing row and reinserted
// with the caller's fields. A caller passing has_voted=false on
// an existing public_key would silently wipe the recorded vote
// and undo the atomic guarantee from TryMarkVoted.
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

	// Mark the voter as having cast a vote (Round 14 atomic primitive).
	require.NoError(t, repo.TryMarkVoted(pk, "vote-hash-abc"))

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
