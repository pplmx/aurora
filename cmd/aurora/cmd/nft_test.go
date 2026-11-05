package cmd

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
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
		assertFieldValue(t, out, "Name:", "Sword")
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
		assertFieldValue(t, gout, "Description:", "d")
		// Optional metadata that was never set must not leave a dangling label.
		assert.NotContains(t, gout, "Image URL:")
		assert.NotContains(t, gout, "Token URI:")
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
		to, toPriv := nftKeypair(t)

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

		// The owner transfers it back through --json: the operation document
		// carries the id, both sides and the block height.
		tout, err = runCmd(t, "nft", "transfer",
			"--nft", id, "--from", to, "--to", pub, "--private-key", toPriv, "--json")
		require.NoError(t, err)
		var op struct {
			ID          string `json:"id"`
			NFTID       string `json:"nft_id"`
			From        string `json:"from"`
			To          string `json:"to"`
			BlockHeight int64  `json:"block_height"`
		}
		require.NoError(t, json.Unmarshal([]byte(tout), &op), "got %q", tout)
		assert.NotEmpty(t, op.ID)
		assert.Equal(t, id, op.NFTID)
		assert.Equal(t, to, op.From)
		assert.Equal(t, pub, op.To)
		assert.Greater(t, op.BlockHeight, int64(0))
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

		// A second asset burns through --json; the document matches the REST
		// endpoint's {"status":"burned"}.
		out, err = runCmd(t, "nft", "mint", "--name", "BurnJSON", "--creator", pub)
		require.NoError(t, err)
		second := extractField(t, out, "ID:")
		require.NotEmpty(t, second)

		out, err = runCmd(t, "nft", "burn",
			"--nft", second, "--owner", pub, "--private-key", priv, "--confirm", "--json")
		require.NoError(t, err)
		var burned struct {
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &burned), "got %q", out)
		assert.Equal(t, "burned", burned.Status)
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
		assert.Contains(t, hout, "Total operations:")
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
		assert.Contains(t, hout, "Total operations: 1")
		assert.Contains(t, hout, "mint")
	})
}

func TestNFTHistory_Pages(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, priv := nftKeypair(t)
		to, _ := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint", "--name", "Paged", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)
		_, err = runCmd(t, "nft", "transfer",
			"--nft", id, "--from", pub, "--to", to, "--private-key", priv)
		require.NoError(t, err)

		hout, err := runCmd(t, "nft", "history", "--nft", id, "--limit", "1", "--offset", "1")
		require.NoError(t, err)
		assert.Contains(t, hout, "Operations on this page: 1 (offset 1)")
		assert.Equal(t, 1, strings.Count(hout, "   - "), "only the requested page should print")

		_, err = runCmd(t, "nft", "history", "--nft", id, "--limit", "-1")
		require.Error(t, err)
		_, err = runCmd(t, "nft", "history", "--nft", id, "--offset", "-1")
		require.Error(t, err)
	})
}

// TestNFTHistory_AfterBurn locks ISS-359 on the persistent CLI path: burn
// removes the NFT row but keeps its audit trail, so history must remain
// available after the destructive operation.
func TestNFTHistory_AfterBurn(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		pub, priv := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint", "--name", "Burned", "--creator", pub)
		require.NoError(t, err)
		id := extractField(t, out, "ID:")
		require.NotEmpty(t, id)

		_, err = runCmd(t, "nft", "burn",
			"--nft", id, "--owner", pub, "--private-key", priv, "--confirm")
		require.NoError(t, err)

		hout, err := runCmd(t, "nft", "history", "--nft", id)
		require.NoError(t, err)
		assert.Contains(t, hout, "Total operations: 2")
		assert.Contains(t, hout, "mint")
		assert.Contains(t, hout, "burn")
	})
}

// TestNFTList_SearchSpansTheStore pins the flag and the two messages it needs.
// The collection is unbounded for the CLI, so "the whole store" is the only
// reading of a term here; what matters is that a term reaches storage at all and
// that a term matching nothing is not reported as an empty collection.
func TestNFTList_SearchSpansTheStore(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
		for _, name := range []string{"Star", "Moon", "Comet"} {
			_, err := runCmd(t, "nft", "mint", "-n", name, "-c", key)
			require.NoError(t, err)
		}

		out, err := runCmd(t, "nft", "list", "--search", "mo")
		require.NoError(t, err)
		assert.Contains(t, out, "Moon")
		assert.NotContains(t, out, "Comet", "a non-match must leave the list")
		assert.Contains(t, out, "Matches: 1",
			"the count names what it counts when a filter is applied")

		noMatch, err := runCmd(t, "nft", "list", "--search", "zzzz")
		require.NoError(t, err)
		assert.Contains(t, noMatch, "Matches: 0")
		assert.Contains(t, noMatch, `No NFT matches "zzzz".`)
		assert.NotContains(t, noMatch, "(none)",
			"a populated collection must not be described as empty")
	})
}

// TestNFTList_RejectsOwnerAndSearchTogether keeps one read to one question.
// Answering the owner and dropping the term printed rows nobody asked for while
// the heading still said "matches".
func TestNFTList_RejectsOwnerAndSearchTogether(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
		_, err := runCmd(t, "nft", "mint", "-n", "Star", "-c", key)
		require.NoError(t, err)

		out, err := runCmd(t, "nft", "list", "--owner", key, "--search", "star")
		require.Error(t, err, "the two flags answer different questions")
		assert.NotContains(t, out, "Star", "and nothing is printed as if it worked")
	})
}

// TestTheNFTListShowsTheHolder pins what a reader needs from the collection.
// The row used to print an id and a name and nothing else, while every other
// list in the CLI carries the fields its module turns on: `token list` shows an
// owner, `oracle source list` a URL and a state. An NFT's holder decides every
// later transfer and burn, so finding it meant reading each asset separately.
func TestTheNFTListShowsTheHolder(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)

		const creator = "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA="
		out, err := runCmd(t, "nft", "mint", "-n", "Star", "-c", creator)
		require.NoError(t, err, "mint")
		nftID := extractField(t, out, "ID:")

		list, err := runCmd(t, "nft", "list")
		require.NoError(t, err)
		require.Contains(t, list, nftID)
		require.Contains(t, list, "Owner:", "the holder must be on the row")
		require.Contains(t, list, creator, "and it must be the actual holder")
		require.Contains(t, list, "Block Height:", "with the block the mint was recorded at")
	})
}

// TestTheNFTHistoryDatesEachOperation pins what a history is read for. The row
// used to print an operation type and a block number, which dates nothing and
// names no holder: an asset's whole past read as a list of block heights.
func TestTheNFTHistoryDatesEachOperation(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)

		const creator = "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA="
		out, err := runCmd(t, "nft", "mint", "-n", "Star", "-c", creator)
		require.NoError(t, err, "mint")
		nftID := extractField(t, out, "ID:")

		hist, err := runCmd(t, "nft", "history", "--nft", nftID)
		require.NoError(t, err)
		require.Contains(t, hist, "mint", "the operation type")
		require.Contains(t, hist, "Block #", "the block it was recorded at")
		// The layout is YYYY-MM-DD, so a row with a date is distinguishable from
		// one with only a block number.
		require.Regexp(t, `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`, hist,
			"a history row must carry the time it happened, not only where")
		require.Contains(t, hist, "Holder after",
			"and a mint names who holds the asset afterwards")
	})
}
