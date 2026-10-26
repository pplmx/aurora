package votingapp

import (
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/require"
)

func TestCastVoteRollbackLeavesNoOnChainBlock(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	db, err := sql.Open("sqlite", sqlite.DSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo, err := sqlite.NewVotingRepository(db)
	require.NoError(t, err)
	chain := blockchain.NewWithDB(db)

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	publicKeyText := base64.StdEncoding.EncodeToString(publicKey)

	voter := voting.NewVoter("Alice")
	voter.PublicKey = publicKeyText
	require.NoError(t, repo.SaveVoter(voter))
	voter.HasVoted = true
	voter.VoteHash = "previous-vote"
	require.NoError(t, repo.UpdateVoter(voter))

	candidate := voting.NewCandidate("Bob", "Independent", "platform")
	require.NoError(t, repo.SaveCandidate(candidate))

	now := time.Now().Unix()
	session := voting.NewSession("Election", "test", []string{candidate.ID}, now-60, now+60)
	require.NoError(t, repo.SaveSession(session))

	uc := NewCastVoteUseCase(repo, voting.NewEd25519Service(), sqlite.NewTxManager(db))
	uc.SetChain(chain)

	_, err = uc.Execute(CastVoteRequest{
		VoterPublicKey: publicKeyText,
		CandidateID:    candidate.ID,
		PrivateKey:     base64.StdEncoding.EncodeToString(privateKey),
		SessionID:      session.ID,
	})
	require.ErrorIs(t, err, voting.ErrAlreadyVoted)
	require.Len(t, chain.Blocks, 1, "losing double vote must not remain in the in-memory chain")

	var blockCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&blockCount))
	require.Zero(t, blockCount, "losing double vote must not persist a block")

	votes, err := repo.GetVotesByCandidate(candidate.ID)
	require.NoError(t, err)
	require.Empty(t, votes)
	storedCandidate, err := repo.GetCandidate(candidate.ID)
	require.NoError(t, err)
	require.Zero(t, storedCandidate.VoteCount)
}
