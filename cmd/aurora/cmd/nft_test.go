package cmd

import (
	"crypto/ed25519"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nftKeypair returns a matching (base64 public key, base64 private key)
// pair. Transfer/burn sign with the private half and the service verifies
// against the public half, so the two MUST come from the same ed25519
// keypair.
func nftKeypair(t *testing.T) (pub, priv string) {
	t.Helper()
	pk := ed25519.PrivateKey(newTestPrivKey(t))
	pubBytes := pk.Public().(ed25519.PublicKey)
	return b64Encode(pubBytes), b64Encode([]byte(pk))
}

func TestNFTMint_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, _ := nftKeypair(t)
		out, err := runCmd(t, "nft", "mint",
			"--name", "Sword", "--description", "rare", "--creator", pub)
		require.NoError(t, err)
		assert.Contains(t, out, "NFT minted successfully!")
		assert.Contains(t, out, "Name: Sword")
	})
}

func TestNFTMint_MissingRequiredFlags(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// missing --creator -> cobra required-flag error.
		_, err := runCmd(t, "nft", "mint", "--name", "X")
		require.Error(t, err)

		// missing --name -> cobra required-flag error.
		pub, _ := nftKeypair(t)
		_, err = runCmd(t, "nft", "mint", "--creator", pub)
		require.Error(t, err)
	})
}

func TestNFTMint_InvalidCreatorBase64(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "nft", "mint",
			"--name", "X", "--creator", "not-base64!!!")
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "invalid creator")
	})
}

func TestNFTGet_Found(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, _ := nftKeypair(t)
		out, err := runCmd(t, "nft", "mint",
			"--name", "GetMe", "--description", "d", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		gout, err := runCmd(t, "nft", "get", "--nft", id)
		require.NoError(t, err)
		assert.Contains(t, gout, "GetMe")
		assert.Contains(t, gout, "NFT Details")
	})
}

func TestNFTGet_NotFound(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "nft", "get", "--nft", "no-such-nft")
		require.Error(t, err)
	})
}

func TestNFTList_Empty(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, _ := nftKeypair(t)
		out, err := runCmd(t, "nft", "list", "--owner", pub)
		require.NoError(t, err)
		assert.Contains(t, out, "(none)")
	})
}

func TestNFTList_Populated(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, _ := nftKeypair(t)
		_, err := runCmd(t, "nft", "mint",
			"--name", "Listed", "--creator", pub)
		require.NoError(t, err)

		out, err := runCmd(t, "nft", "list", "--owner", pub)
		require.NoError(t, err)
		assert.Contains(t, out, "NFTs owned: 1")
		assert.Contains(t, out, "Listed")
	})
}

func TestNFTTransfer_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, priv := nftKeypair(t)
		to, _ := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint",
			"--name", "Item", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		tout, err := runCmd(t, "nft", "transfer",
			"--nft", id, "--from", pub, "--to", to, "--private-key", priv)
		require.NoError(t, err)
		assert.Contains(t, tout, "NFT transferred successfully!")
		assert.Contains(t, tout, "To:")
	})
}

func TestNFTTransfer_WrongKey(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, _ := nftKeypair(t)
		wrongPriv, _ := nftKeypair(t) // different keypair — invalid signature
		to, _ := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint",
			"--name", "Item2", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		_, err = runCmd(t, "nft", "transfer",
			"--nft", id, "--from", pub, "--to", to, "--private-key", wrongPriv)
		require.Error(t, err)
	})
}

func TestNFTBurn_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, priv := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint",
			"--name", "BurnMe", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		bout, err := runCmd(t, "nft", "burn",
			"--nft", id, "--owner", pub, "--private-key", priv, "--confirm")
		require.NoError(t, err)
		assert.Contains(t, bout, "NFT burned successfully!")
	})
}

func TestNFTBurn_RequiresConfirm(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, priv := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint",
			"--name", "BurnFail", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		_, err = runCmd(t, "nft", "burn",
			"--nft", id, "--owner", pub, "--private-key", priv)
		require.Error(t, err, "burn without --confirm must be refused")
		assert.Contains(t, err.Error(), "--confirm")
	})
}

func TestNFTBurn_WrongKey(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, _ := nftKeypair(t)
		wrongPriv, _ := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint",
			"--name", "BurnFail", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		_, err = runCmd(t, "nft", "burn",
			"--nft", id, "--owner", pub, "--private-key", wrongPriv, "--confirm")
		require.Error(t, err)
	})
}

func TestNFTHistory_Populated(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, priv := nftKeypair(t)
		to, _ := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint",
			"--name", "Hist", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		_, err = runCmd(t, "nft", "transfer",
			"--nft", id, "--from", pub, "--to", to, "--private-key", priv)
		require.NoError(t, err)

		hout, err := runCmd(t, "nft", "history", "--nft", id)
		require.NoError(t, err)
		assert.Contains(t, hout, "Operations:")
	})
}

// TestNFTHistory_ShowsMintOp: the "(none)" branch is unreachable through
// the CLI because mint itself records the first operation, so history for
// any CLI-minted NFT shows at least the mint op.
func TestNFTHistory_ShowsMintOp(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, _ := nftKeypair(t)
		out, err := runCmd(t, "nft", "mint",
			"--name", "NoOp", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		hout, err := runCmd(t, "nft", "history", "--nft", id)
		require.NoError(t, err)
		assert.Contains(t, hout, "Operations: 1")
		assert.Contains(t, hout, "mint")
	})
}
