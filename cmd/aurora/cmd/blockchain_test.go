package cmd

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/ledger"
	"github.com/pplmx/aurora/internal/records"
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
		assert.Regexp(t, `Head: [0-9a-f]{12}`, out,
			"the head prefix is what a support thread compares between machines")
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
// first write, and a recorded draw appears as the decoded record the window's
// ledger page also shows (`--json` keeps the exact payload for a script).
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
		assert.Contains(t, out, "lottery · Draw")
		assert.Contains(t, out, "Seed: demo-seed")

		// The filter matches the recorded text regardless of case, and a miss
		// says so instead of claiming the ledger is empty.
		out, err = runCmd(t, "blockchain", "blocks", "-s", "DEMO-SEED")
		require.NoError(t, err)
		assert.Contains(t, out, "Audit records: 1")
		assert.Contains(t, out, "Seed: demo-seed")

		// The filter reaches the decoded sentence, not only the stored text, so
		// a word the table shows ("Winners") finds the record.
		out, err = runCmd(t, "blockchain", "blocks", "-s", "Winners")
		require.NoError(t, err)
		assert.Contains(t, out, "Audit records: 1")

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

// TestBlockLineEncodesAnUnrecognisedPayload covers the table's fallback. A
// ledger payload matching no known shape has no fields to sanitise, so the
// whole value is what the row prints. Returning it unchanged let a record
// carrying an escape sequence recolour the table or split it with a newline;
// --json still carries the exact bytes.
func TestBlockLineEncodesAnUnrecognisedPayload(t *testing.T) {
	for _, raw := range []string{
		"forged \x1b[31mRED\x1b[0m row",
		"forged\nsecond line",
	} {
		got := blockLine(raw)
		assert.NotEqualf(t, raw, got, "an unprintable payload must not print as-is")
		assert.Truef(t, records.IsPrintable(got), "the table line must be printable: %q", got)
	}

	// A recognised payload still renders as its sentence.
	assert.Equal(t, "token · Mint · GLD   (Token: GLD; To: dXNlcg==)",
		blockLine("mint|GLD|dXNlcg=="))
}

// TestBlockLineTranslatesTheRecordVocabulary covers the ledger table in a
// Chinese process: the row is a sentence a reader scans, and its module, action
// and fact labels came out in English while the heading above them was Chinese.
//
// The punctuation moves too. A translated row printed 「抽奖 · C, A   (中奖者:
// C, A)」 — half-width brackets, colon and semicolons inside Chinese copy.
func TestBlockLineTranslatesTheRecordVocabulary(t *testing.T) {
	const payload = `{"participants":["A","B"],"seed":"s","winners":["A"]}`

	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("en")
	require.Equal(t, `lottery · Draw · A   (Winners: A; Participants: 2; Seed: s)`, blockLine(payload))

	tr.SetLocale("zh")
	require.Equal(t, `抽奖 · A（中奖者：A；参与者：2；种子：s）`, blockLine(payload))

	// A payload the decoder does not understand is still escaped, in either
	// language: the fallback never renders raw bytes.
	tr.SetLocale("zh")
	require.True(t, records.IsPrintable(blockLine("forged \x1b[31mRED\x1b[0m")))
}

// TestBlockMatchesSearchesTheEnglishWords covers the filter in a Chinese
// process. Once a row is translated, a reader who types the word the
// documentation and the JSON use — "Winners" — must still find the record, and
// so must the reader who types 中奖者 after reading it on screen.
//
// Only the current language and the canonical English are searched; an English
// window does not answer a Chinese query, which is the tradeoff that keeps the
// filter from carrying every catalog.
func TestBlockMatchesSearchesTheEnglishWords(t *testing.T) {
	entry := ledger.BlockEntry{
		Height: 2,
		Data:   `{"participants":["A","B"],"seed":"s","winners":["A"]}`,
	}

	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("zh")
	for _, query := range []string{"winners", "中奖者", "lottery", "抽奖", "seed"} {
		require.Truef(t, blockMatches(entry, strings.ToLower(query)),
			"a Chinese window must find the record by %q", query)
	}
	require.False(t, blockMatches(entry, "no-such-word"))

	tr.SetLocale("en")
	for _, query := range []string{"winners", "lottery", "draw"} {
		require.Truef(t, blockMatches(entry, query), "an English window must find the record by %q", query)
	}
}

// TestJSONListCommandsPrintTheDocumentedOuterShape pins the "list commands print
// an array" promise in docs/cli.md, and the one exception beside it.
//
// The reference used to put `blockchain blocks` in the array row: it prints a
// {"blocks","total","limit","offset"} page, so a script that indexed the output
// as a list read null. Both facts are cheap to check and neither is visible from
// the command running successfully.
func TestJSONListCommandsPrintTheDocumentedOuterShape(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		// token history needs a token that exists; the rest print [] on an
		// empty database, which is the case a script hits first.
		_, err := runCmd(t, "token", "create", "-n", "Gold", "-s", "GLD", "--supply", "1000")
		require.NoError(t, err)

		arrayCommands := [][]string{
			{"lottery", "history"},
			{"token", "list"},
			{"token", "history", "--token", "GLD"},
			{"nft", "list"},
			{"voting", "candidate", "list"},
			{"voting", "voter", "list"},
			{"voting", "session", "list"},
			{"oracle", "source", "list"},
			{"oracle", "template", "list"},
		}
		for _, args := range arrayCommands {
			out, err := runCmd(t, append(append([]string{}, args...), "--json")...)
			require.NoErrorf(t, err, "%v must succeed on an empty database", args)
			var probe any
			require.NoErrorf(t, json.Unmarshal([]byte(out), &probe), "%v must print JSON", args)
			_, isArray := probe.([]any)
			assert.Truef(t, isArray, "%v must print an array, got %T (docs/cli.md's list row)", args, probe)
		}

		out, err := runCmd(t, "blockchain", "blocks", "--json")
		require.NoError(t, err)
		var page map[string]any
		require.NoError(t, json.Unmarshal([]byte(out), &page), "blocks must print one document")
		for _, key := range []string{"blocks", "total", "limit", "offset"} {
			assert.Containsf(t, page, key, "the page must carry %q", key)
		}
		// The empty ledger is `[]`, never null: a script that ranges over it
		// would otherwise have to special-case the first run of a fresh install.
		blocks, ok := page["blocks"].([]any)
		require.True(t, ok, "blocks must be an array")
		assert.Empty(t, blocks)
	})
}
