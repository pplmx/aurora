package votingapp

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCastVoteUseCase_AllowsOneBallotPerVoterPerSession pins the election
// boundary: HasVoted is a lifetime profile flag, not a lifetime ballot lock.
// A registered voter may cast one ballot in each active session.
func TestCastVoteUseCase_AllowsOneBallotPerVoterPerSession(t *testing.T) {
	repo, txMgr, cleanup := setupCastVoteTestDB(t)
	defer cleanup()

	service := voting.NewEd25519Service()
	now := time.Now().Unix()
	candidate := voting.NewCandidate("Alice", "Party A", "Platform")
	require.NoError(t, repo.SaveCandidate(candidate))

	session1 := voting.NewSession("Election 2026", "", []string{candidate.ID}, now-3600, now+3600)
	session1.Status = voting.SessionStatusActive
	require.NoError(t, repo.SaveSession(session1))

	session2 := voting.NewSession("Election 2027", "", []string{candidate.ID}, now-3600, now+3600)
	session2.Status = voting.SessionStatusActive
	require.NoError(t, repo.SaveSession(session2))

	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)
	require.NoError(t, repo.SaveVoter(&voting.Voter{
		PublicKey: pubB64, Name: "Alice", RegisteredAt: now,
	}))

	cast := func(sessionID string) error {
		_, err := NewCastVoteUseCase(repo, service, txMgr).Execute(CastVoteRequest{
			VoterPublicKey: pubB64,
			CandidateID:    candidate.ID,
			PrivateKey:     privB64,
			SessionID:      sessionID,
		})
		return err
	}

	require.NoError(t, cast(session1.ID), "first session ballot")
	require.NoError(t, cast(session2.ID), "the same voter must be eligible in the next session")
	require.ErrorIs(t, cast(session2.ID), voting.ErrAlreadyVoted,
		"a session still accepts only one ballot from the same voter")

	voter, err := repo.GetVoter(pubB64)
	require.NoError(t, err)
	assert.True(t, voter.HasVoted, "the profile flag records that this voter has ever voted")

	got, err := repo.GetCandidate(candidate.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.VoteCount)

	first, err := repo.CountVotesBySession(session1.ID, candidate.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, first)
	second, err := repo.CountVotesBySession(session2.ID, candidate.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, second)

	votes, err := repo.GetVotesByVoter(pubB64)
	require.NoError(t, err)
	assert.Len(t, votes, 2)
}
