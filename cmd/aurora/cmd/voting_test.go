package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// votingFixture wires a candidate + session + voter via the CLI and returns
// their ids/keys so a test can drive `voting vote`. The real migrations are
// applied first so the CLI's lazy DB matches a properly-initialised install.
func votingFixture(t *testing.T) (candID, sessionID, voterPub, voterPriv string) {
	t.Helper()
	runMigrations(t)

	out, err := runCmd(t, "voting", "candidate", "add", "--name", "Alice", "--party", "P")
	require.NoError(t, err, "candidate add")
	candID = extractField(t, out, "ID:")
	require.NotEmpty(t, candID)

	sout, err := runCmd(t, "voting", "session", "create",
		"--title", "Election 2026", "--candidates", candID,
		"--start-time", "1700000000", "--end-time", "9999999999")
	require.NoError(t, err, "session create")
	sessionID = extractField(t, sout, "ID:")
	require.NotEmpty(t, sessionID)

	vout, err := runCmd(t, "voting", "voter", "register", "--name", "Bob")
	require.NoError(t, err, "voter register")
	voterPub = extractKey(t, vout, "Public Key")
	voterPriv = extractKey(t, vout, "Private Key")
	require.NotEmpty(t, voterPub)
	require.NotEmpty(t, voterPriv)

	return candID, sessionID, voterPub, voterPriv
}

func TestVotingCandidateAdd_List(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		out, err := runCmd(t, "voting", "candidate", "add", "--name", "Alice", "--party", "P")
		require.NoError(t, err)
		assert.Contains(t, out, "Candidate registered: Alice")
		assert.Contains(t, out, "Party: P")

		out, err = runCmd(t, "voting", "candidate", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "Alice")
		assert.Contains(t, out, "- 0 votes")
	})
}

func TestVotingCandidateList_Empty(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		out, err := runCmd(t, "voting", "candidate", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "(none)")
	})
}

func TestVotingCandidateAdd_MissingName(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		_, err := runCmd(t, "voting", "candidate", "add", "--party", "P")
		require.Error(t, err)
	})
}

func TestVotingVoterRegister_List(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		out, err := runCmd(t, "voting", "voter", "register", "--name", "Carol")
		require.NoError(t, err)
		assert.Contains(t, out, "Voter registered successfully!")
		assert.NotEmpty(t, extractKey(t, out, "Public Key"))

		out, err = runCmd(t, "voting", "voter", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "Carol")
		assert.Contains(t, out, "not voted")
	})
}

func TestVotingVoterRegister_MissingName(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		_, err := runCmd(t, "voting", "voter", "register")
		require.Error(t, err)
	})
}

func TestVotingVote_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		candID, sessionID, voterPub, voterPriv := votingFixture(t)

		out, err := runCmd(t, "voting", "vote",
			"--voter", voterPub, "--candidate", candID,
			"--private-key", voterPriv, "--session", sessionID)
		require.NoError(t, err)
		assert.Contains(t, out, "Vote cast successfully!")
		assert.Contains(t, out, "Vote ID:")
	})
}

func TestVotingVote_AlreadyVoted(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		candID, sessionID, voterPub, voterPriv := votingFixture(t)

		_, err := runCmd(t, "voting", "vote",
			"--voter", voterPub, "--candidate", candID,
			"--private-key", voterPriv, "--session", sessionID)
		require.NoError(t, err)

		// Same voter voting again must be rejected (double-vote protection).
		_, err = runCmd(t, "voting", "vote",
			"--voter", voterPub, "--candidate", candID,
			"--private-key", voterPriv, "--session", sessionID)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "already voted")
	})
}

func TestVotingVote_UnregisteredVoter(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		candID, sessionID, _, _ := votingFixture(t)

		// A keypair that was never registered as a voter.
		ghostPub, ghostPriv := nftKeypair(t)

		_, err := runCmd(t, "voting", "vote",
			"--voter", ghostPub, "--candidate", candID,
			"--private-key", ghostPriv, "--session", sessionID)
		require.Error(t, err)
		// The VoterRepository surfaces a miss as ErrNotFound, which the
		// CLI wraps as "failed to get voter: record not found".
		assert.Contains(t, strings.ToLower(err.Error()), "voter")
		assert.Contains(t, strings.ToLower(err.Error()), "not found")
	})
}

func TestVotingSessionCreate_List_Start_End(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		candID, sessionID, _, _ := votingFixture(t)
		_ = candID

		// list shows the session
		out, err := runCmd(t, "voting", "session", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "Election 2026")

		// start then end transition status; the CLI prints confirmation
		out, err = runCmd(t, "voting", "session", "start", "--id", sessionID)
		require.NoError(t, err)
		assert.Contains(t, out, "Session started!")

		out, err = runCmd(t, "voting", "session", "end", "--id", sessionID)
		require.NoError(t, err)
		assert.Contains(t, out, "Session ended!")
	})
}

func TestVotingSessionCreate_InvalidTimes(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		out, err := runCmd(t, "voting", "candidate", "add", "--name", "X", "--party", "P")
		require.NoError(t, err)
		cid := extractField(t, out, "ID:")
		require.NotEmpty(t, cid)

		// end <= start is rejected by the usecase.
		_, err = runCmd(t, "voting", "session", "create",
			"--title", "E", "--candidates", cid,
			"--start-time", "200", "--end-time", "100")
		require.Error(t, err)
	})
}

func TestVotingSessionStart_NotFound(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		_, err := runCmd(t, "voting", "session", "start", "--id", "no-such")
		require.Error(t, err)
	})
}

func TestVotingResults_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		candID, sessionID, voterPub, voterPriv := votingFixture(t)

		_, err := runCmd(t, "voting", "vote",
			"--voter", voterPub, "--candidate", candID,
			"--private-key", voterPriv, "--session", sessionID)
		require.NoError(t, err)

		out, err := runCmd(t, "voting", "results", "--session", sessionID)
		require.NoError(t, err)
		assert.Contains(t, out, "Results:")
		assert.Contains(t, out, "Alice")
		assert.Contains(t, out, "1 votes")
	})
}

func TestVotingResults_NotFound(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		_, err := runCmd(t, "voting", "results", "--session", "no-such")
		require.Error(t, err)
	})
}
