package deskapp

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/app/tokenapp"
	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/stretchr/testify/require"
)

// TestGenerateKeyPairIsAUsableEd25519Pair checks the one service the desktop
// added for its forms: the two halves must be the right size and belong
// together, because a token owner key that cannot sign would only fail later,
// on the first mint.
func TestGenerateKeyPairIsAUsableEd25519Pair(t *testing.T) {
	services := &nativeServices{}
	pubText, privText, err := services.GenerateKeyPair()
	require.NoError(t, err)

	pub, err := base64.StdEncoding.DecodeString(pubText)
	require.NoError(t, err)
	require.Len(t, pub, ed25519.PublicKeySize)

	priv, err := base64.StdEncoding.DecodeString(privText)
	require.NoError(t, err)
	require.Len(t, priv, ed25519.PrivateKeySize)

	message := []byte("aurora")
	signature := ed25519.Sign(ed25519.PrivateKey(priv), message)
	require.True(t, ed25519.Verify(ed25519.PublicKey(pub), message, signature),
		"the public half must verify what the private half signs")
}

// TestHealthRowsCarrySchedulerCounters pins the adapter the Feed Health card
// reads. It used to return (nil, nil) unconditionally, so the card could only
// ever say "(no feed activity yet)" while the REST endpoint showed counters.
func TestHealthRowsCarrySchedulerCounters(t *testing.T) {
	row := healthRow(oracleapp.SourceStat{
		SourceID: "s1", Attempts: 7, Successes: 5, Failures: 2,
		LastSuccessAt: 1_700_000_000, LastError: "upstream 503",
	})
	require.Equal(t, "s1", row.SourceID)
	require.Equal(t, 7, row.Attempts)
	require.Equal(t, 5, row.Successes)
	require.Equal(t, 2, row.Failures)
	require.Equal(t, int64(1_700_000_000), row.LastSuccess)
	require.Equal(t, "upstream 503", row.LastError)
}

// TestOracleHealthWithoutASchedulerIsEmpty keeps the adapter nil-safe: a test
// or a runtime built before StartOracleScheduler must report no activity, not
// panic on a nil provider.
func TestOracleHealthWithoutASchedulerIsEmpty(t *testing.T) {
	rows, err := (&nativeServices{}).OracleHealth()
	require.NoError(t, err)
	require.Empty(t, rows)
}

// TestTransferRowsCarryNonces pins the second half of the token history
// mapping: the use case now returns a nonce, and the view row must keep it or
// the desktop column still renders 0.
func TestTransferRowsCarryNonces(t *testing.T) {
	row := transferRow(tokenapp.TransferResponse{
		From: "alice", To: "bob", Amount: "12", Nonce: 9,
	})
	require.Equal(t, "alice", row.From)
	require.Equal(t, "bob", row.To)
	require.Equal(t, "12", row.Amount)
	require.Equal(t, uint64(9), row.Nonce)
}

// TestSessionRowsCarryDescriptions pins the field the Create Session form
// collects and the selected-session preview shows. The adapter dropped it, so
// the text was stored but unreachable from the window.
func TestSessionRowsCarryDescriptions(t *testing.T) {
	row := sessionRow(&votingapp.SessionResponse{
		ID: "s1", Title: "Budget", Description: "Approve 2027",
		Status: "active", Candidates: []string{"c1"}, StartTime: 10, EndTime: 20,
	})
	require.Equal(t, "s1", row.ID)
	require.Equal(t, "Budget", row.Title)
	require.Equal(t, "Approve 2027", row.Description)
	require.Equal(t, "active", row.Status)
	require.Equal(t, []string{"c1"}, row.Candidates)
	require.Equal(t, int64(10), row.StartTime)
	require.Equal(t, int64(20), row.EndTime)
}

// TestLedgerBlocksCarryBlockHeights pins the heights the desktop ledger shows.
// The adapter used each payload's index in the filtered chain, so the first
// recorded block displayed height 0 and every row was one too low.
func TestLedgerBlocksCarryBlockHeights(t *testing.T) {
	chain := blockchain.NewBlockChain()
	_, err := chain.AddBlock(`{"module":"lottery"}`)
	require.NoError(t, err)
	_, err = chain.AddBlock(`{"module":"voting"}`)
	require.NoError(t, err)

	services := &nativeServices{chain: chain}
	rows, err := services.LedgerBlocks(0)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, int64(2), rows[0].Height, "the newest block is at height 2")
	require.Equal(t, "voting", rows[0].Module)
	require.Equal(t, int64(1), rows[1].Height)
	require.Equal(t, "lottery", rows[1].Module)

	latest, err := services.LedgerBlocks(1)
	require.NoError(t, err)
	require.Len(t, latest, 1)
	require.Equal(t, int64(2), latest[0].Height)
}
