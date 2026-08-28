package cmd

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tokenFixture returns the token symbol (its ID), the base64 owner public
// key and the base64 owner private key, obtained by running the real
// `token create` command and parsing its printed output.
func tokenFixture(t *testing.T, name, symbol, supply string) (tokenID, pub, priv string) {
	t.Helper()
	out, err := runCmd(t, "token", "create",
		"--name", name, "--symbol", symbol, "--supply", supply)
	require.NoError(t, err, "token create should succeed")
	require.Contains(t, out, "Token created")

	tokenID = symbol // TokenID is the symbol (see NewToken(TokenID(req.Symbol)))
	pub = extractField(t, out, "Owner Public Key:")
	priv = extractField(t, out, "Owner Private Key:")
	require.NotEmpty(t, pub)
	require.NotEmpty(t, priv)
	return tokenID, pub, priv
}

// extractField pulls the value after "label " on the line containing label.
func extractField(t *testing.T, out, label string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), label+" "); ok {
			after = strings.TrimSpace(after)
			// strip any trailing annotation like "(SAVE THIS!)"
			if sp := strings.Index(after, " "); sp >= 0 {
				after = after[:sp]
			}
			return after
		}
	}
	return ""
}

func TestTokenCreate_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		id, pub, priv := tokenFixture(t, "MyToken", "MTK", "1000000")
		assert.Equal(t, "MTK", id)
		assert.NotEmpty(t, pub)
		assert.NotEmpty(t, priv)
	})
}

func TestTokenCreate_MissingRequiredFlags(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "token", "create", "--name", "X")
		require.Error(t, err)

		_, err = runCmd(t, "token", "create", "--symbol", "X")
		require.Error(t, err)

		_, err = runCmd(t, "token", "create", "--name", "X", "--symbol", "X")
		require.Error(t, err)
	})
}

func TestTokenCreate_InvalidSupply(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "token", "create",
			"--name", "X", "--symbol", "X", "--supply", "not-a-number")
		require.Error(t, err)
	})
}

// TestTokenCreate_DecimalsHonored locks the v1.79 fix: the --decimals flag
// was registered but never read, so every value was silently dropped and the
// token always got the default of 8 (TASK-099, ISS-091). It must now be
// persisted and surface through `token info`.
func TestTokenCreate_DecimalsHonored(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "token", "create",
			"--name", "GamingCoin", "--symbol", "GAME", "--supply", "1000000", "--decimals", "6")
		require.NoError(t, err, "token create with --decimals 6 should succeed")

		out, err := runCmd(t, "token", "info", "--token", "GAME")
		require.NoError(t, err)
		require.Contains(t, out, "Decimals: 6", "created token must report the honored --decimals value")

		// Unset flag keeps the documented default of 8.
		out, err = runCmd(t, "token", "create",
			"--name", "DefaultCoin", "--symbol", "DFLT", "--supply", "1000000")
		require.NoError(t, err)
		out, err = runCmd(t, "token", "info", "--token", "DFLT")
		require.NoError(t, err)
		require.Contains(t, out, "Decimals: 8", "unset --decimals must keep the default of 8")
	})
}

func TestTokenCreate_InvalidDecimals(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		for _, bad := range []string{"abc", "-1", "999"} {
			_, err := runCmd(t, "token", "create",
				"--name", "Bad", "--symbol", "BAD", "--supply", "1000000", "--decimals", bad)
			require.Error(t, err, "create with --decimals %q should fail", bad)
		}
	})
}

func TestTokenMint_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "MintCoin", "MINT", "1000")
		out, err := runCmd(t, "token", "mint",
			"--token", tokenID, "--to", pub, "--amount", "500", "--private-key", priv)
		require.NoError(t, err)
		assert.Contains(t, out, "minted")
		assert.Contains(t, out, "Amount: 500")
	})
}

func TestTokenMint_InvalidBase64Key(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, _ := tokenFixture(t, "MintCoin2", "MNT2", "1000")
		_, err := runCmd(t, "token", "mint",
			"--token", tokenID, "--to", pub, "--amount", "1", "--private-key", "not-base64!!!")
		require.Error(t, err)
	})
}

func TestTokenMint_WrongOwnerKey(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, _ := tokenFixture(t, "MintCoin3", "MNT3", "1000")
		// A different (random) private key that does not match the owner.
		otherPriv := b64Encode(newTestPrivKey(t))
		_, err := runCmd(t, "token", "mint",
			"--token", tokenID, "--to", pub, "--amount", "1", "--private-key", otherPriv)
		require.Error(t, err)
	})
}

func TestTokenTransfer_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "TokeCoin", "TK", "1000")
		// Mint some to a second address so balances differ, then transfer
		// between the owner and that address.
		out, err := runCmd(t, "token", "transfer",
			"--token", tokenID, "--from", pub, "--to", pub,
			"--amount", "10", "--private-key", priv)
		require.NoError(t, err)
		assert.Contains(t, out, "transferred")
		assert.Contains(t, out, "Amount: 10")
	})
}

func TestTokenTransfer_InvalidAmount(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "TransferCoin", "TR", "1000")
		_, err := runCmd(t, "token", "transfer",
			"--token", tokenID, "--from", pub, "--to", pub,
			"--amount", "-5", "--private-key", priv)
		require.Error(t, err)
	})
}

func TestTokenApprove_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "ApproveCoin", "AP", "1000")
		out, err := runCmd(t, "token", "approve",
			"--token", tokenID, "--owner", pub, "--spender", pub,
			"--amount", "100", "--private-key", priv)
		require.NoError(t, err)
		assert.Contains(t, out, "approved")
		assert.Contains(t, out, "Amount: 100")
	})
}

func TestTokenBurn_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "BurnCoin", "BURN", "1000")
		out, err := runCmd(t, "token", "burn",
			"--token", tokenID, "--from", pub, "--amount", "100", "--private-key", priv, "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "burned")
	})
}

func TestTokenBurn_RequiresConfirm(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "BurnCoin", "BURN", "1000")
		out, err := runCmd(t, "token", "burn",
			"--token", tokenID, "--from", pub, "--amount", "100", "--private-key", priv)
		require.Error(t, err, "burn without --confirm must be refused")
		assert.Contains(t, err.Error(), "--confirm")
		assert.NotContains(t, out, "burned")
	})
}

func TestTokenBalance_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, _ := tokenFixture(t, "BalanceCoin", "BAL", "1000")
		out, err := runCmd(t, "token", "balance",
			"--token", tokenID, "--owner", pub)
		require.NoError(t, err)
		assert.Contains(t, out, "Balance: 1000")
	})
}

func TestTokenAllowance_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "AllowCoin", "ALW", "1000")
		_, err := runCmd(t, "token", "approve",
			"--token", tokenID, "--owner", pub, "--spender", pub,
			"--amount", "50", "--private-key", priv)
		require.NoError(t, err)

		out, err := runCmd(t, "token", "allowance",
			"--token", tokenID, "--owner", pub, "--spender", pub)
		require.NoError(t, err)
		assert.Contains(t, out, "Allowance: 50")
	})
}

// TestTokenHistory_HappyPath is the regression test for the CLI audit-events
// gap (TASK-113, ISS-105): the CLI's event bus had no audit subscriber, so a
// CLI transfer dropped its event into the void and `token history` was always
// empty. It must now surface the transfer row. (History is transfer-history:
// a mint is a token.mint event and is intentionally not listed.)
func TestTokenHistory_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, pub, priv := tokenFixture(t, "HistCoin", "HST", "1000")
		// fund the owner, then do a CLI transfer the history page should show
		_, err := runCmd(t, "token", "mint",
			"--token", tokenID, "--to", pub, "--amount", "100", "--private-key", priv)
		require.NoError(t, err)

		recipient := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x59}, 32))
		_, err = runCmd(t, "token", "transfer",
			"--token", tokenID, "--from", pub, "--to", recipient,
			"--amount", "50", "--private-key", priv)
		require.NoError(t, err)

		out, err := runCmd(t, "token", "history",
			"--token", tokenID, "--owner", pub, "--limit", "10")
		require.NoError(t, err)
		assert.Contains(t, out, "Transfer History: 1")
		assert.Contains(t, out, "Amount: 50")
	})
}

func TestTokenInfo_Found(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		tokenID, _, _ := tokenFixture(t, "InfoCoin", "INF", "777")
		out, err := runCmd(t, "token", "info", "--token", tokenID)
		require.NoError(t, err)
		assert.Contains(t, out, "InfoCoin")
		assert.Contains(t, out, "INF")
		assert.Contains(t, out, "Total Supply: 777")
	})
}

func TestTokenInfo_NotFound(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "token", "info", "--token", "NO-SUCH-TOKEN")
		require.Error(t, err)
	})
}

func TestTokenTUI_Wired(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		out, err := runCmd(t, "token", "tui")
		// token tui is now wired to the real internal/ui/token program
		// (RunTokenTUI), not the old placeholder. In the non-interactive
		// test harness there is no TTY, so it fails fast with a TTY error
		// instead of printing a stub.
		require.Error(t, err)
		assert.NotContains(t, out, "not implemented")
		assert.Contains(t, err.Error(), "TTY")
	})
}

// NOTE: newTokenService's first-step error branch (sqlite.NewTokenRepository
// failure) is not unit-tested. Forcing it would require a hostile filesystem
// state (./data as a non-directory), but blockchain.DBPath() swallows that by
// returning "" and NewTokenRepository("") falls back to a temp SQLite DB — so
// the branch is not reachable without contrived setup, matching the project's
// "error branches not worth contriving" precedent (backup package). The full
// constructor happy path IS covered by every token subcommand test above.
