package deskapp

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/pplmx/aurora/internal/domain/blockchain"
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
