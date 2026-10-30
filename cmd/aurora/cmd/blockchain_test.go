package cmd

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/infra/blockchain"
)

// TestBlockchainVerifyReportsAValidChain: a fresh database holds only the
// genesis block, and the command must report it as valid and exit zero.
func TestBlockchainVerifyReportsAValidChain(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		out, err := runCmd(t, "blockchain", "verify")
		require.NoError(t, err)
		assert.Contains(t, out, "Chain integrity")
		assert.Contains(t, out, "Blocks: 1")
		assert.Contains(t, out, "State: valid")
	})
}

// TestBlockchainVerifyFailsOnTamperedData: the exit code is the machine-readable
// verdict, so persisted data that no longer matches its hash must fail the
// command. The load path may refuse the chain before the report is built (a
// corrupted genesis), which is also an error, so the test asserts the exit code
// rather than one message.
func TestBlockchainVerifyFailsOnTamperedData(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")
		// Keep the database outside ./data: ResetForTest removes that directory
		// and would delete the file this test needs to tamper with.
		dbPath := filepath.Join(t.TempDir(), "chain.db")
		before := viper.GetString("db.path")
		viper.Set("db.path", dbPath)
		t.Cleanup(func() { viper.Set("db.path", before) })

		_, err := runCmd(t, "blockchain", "verify")
		require.NoError(t, err)

		blockchain.ResetForTest()
		t.Cleanup(blockchain.ResetForTest)

		db, err := sql.Open("sqlite", dbPath)
		require.NoError(t, err)
		_, err = db.Exec("UPDATE blocks SET data = 'tampered' WHERE height = 0")
		require.NoError(t, err)
		require.NoError(t, db.Close())

		_, err = runCmd(t, "blockchain", "verify")
		require.Error(t, err)
	})
}

// TestBlockchainBlocksListsRecordedPayloads: the empty ledger names the missing
// first write, and a recorded draw appears with its exact payload so a script
// can grep the audit trail.
func TestBlockchainBlocksListsRecordedPayloads(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		out, err := runCmd(t, "blockchain", "blocks")
		require.NoError(t, err)
		assert.Contains(t, out, "No module has written a record yet.")

		_, err = runCmd(t, "lottery", "create",
			"--participants", "A,B,C", "--seed", "demo-seed", "--count", "1")
		require.NoError(t, err)

		out, err = runCmd(t, "blockchain", "blocks")
		require.NoError(t, err)
		assert.Contains(t, out, "Audit records: 1")
		assert.Contains(t, out, `"seed":"demo-seed"`)
	})
}
