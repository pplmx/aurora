package cmd

import (
	"database/sql"
	"encoding/json"
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

		// JSON mode fails the command too. A chain refused at load has no
		// report to print, so this asserts the exit code, not a document.
		_, err = runCmd(t, "blockchain", "verify", "--json")
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

		// The filter matches the recorded text regardless of case, and a miss
		// says so instead of claiming the ledger is empty.
		out, err = runCmd(t, "blockchain", "blocks", "-s", "DEMO-SEED")
		require.NoError(t, err)
		assert.Contains(t, out, "Audit records: 1")
		assert.Contains(t, out, `"seed":"demo-seed"`)

		out, err = runCmd(t, "blockchain", "blocks", "-s", "no-such-record")
		require.NoError(t, err)
		assert.Contains(t, out, "Audit records: 0")
		assert.Contains(t, out, `No record matches "no-such-record"`)

		// An offset past the last entry is a paging problem, not an empty ledger.
		out, err = runCmd(t, "blockchain", "blocks", "-p", "5")
		require.NoError(t, err)
		assert.Contains(t, out, "Audit records: 1")
		assert.Contains(t, out, "No entries on this page")
	})
}

// TestBlockchainJSONMatchesTheAPI turns the two commands into something a script
// can parse: the verify report and the block page use the REST shapes, and the
// document starts at the first byte (no banner to strip).
func TestBlockchainJSONMatchesTheAPI(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		out, err := runCmd(t, "blockchain", "verify", "--json")
		require.NoError(t, err)
		var report struct {
			Valid  bool `json:"valid"`
			Length int  `json:"length"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &report), "output must be one JSON document: %q", out)
		assert.True(t, report.Valid)
		assert.Equal(t, 1, report.Length)

		_, err = runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "json-seed", "--count", "1")
		require.NoError(t, err)

		out, err = runCmd(t, "blockchain", "blocks", "--json")
		require.NoError(t, err)
		var page struct {
			Blocks []struct {
				Height  int64  `json:"height"`
				Payload string `json:"payload"`
			} `json:"blocks"`
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &page), "output must be one JSON document: %q", out)
		assert.Equal(t, 1, page.Total)
		require.Len(t, page.Blocks, 1)
		assert.Equal(t, int64(1), page.Blocks[0].Height)
		assert.Contains(t, page.Blocks[0].Payload, "json-seed")
	})
}
