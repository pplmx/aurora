package deskapp

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

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
