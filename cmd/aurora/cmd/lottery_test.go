package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainlottery "github.com/pplmx/aurora/internal/domain/lottery"
)

func TestLotteryCreate_HappyPath(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		out, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob,Charlie", "--seed", "lottery-seed-1", "--count", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Winners:")
		assert.Contains(t, out, "Lottery ID:")
		assert.Contains(t, out, "Block height:")
	})
}

func TestLotteryCreate_MissingRequiredFlags(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// Missing --participants -> cobra required-flag validation error.
		_, err := runCmd(t, "lottery", "create", "--seed", "seed")
		require.Error(t, err)

		// Missing --seed.
		_, err = runCmd(t, "lottery", "create", "--participants", "A,B")
		require.Error(t, err)

		// Nothing at all.
		_, err = runCmd(t, "lottery", "create")
		require.Error(t, err)
	})
}

func TestLotteryHistory_Empty(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		out, err := runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, out, "No lottery records found")
	})
}

func TestLotteryHistory_AfterCreate(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "history-seed", "--count", "1")
		require.NoError(t, err)

		out, err := runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, out, "Total lotteries: 1")
		assert.Contains(t, out, "--- Lottery #1 ---")
	})
}

func TestLotteryVerify_ByHeight_Found(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob,Carol", "--seed", "verify-seed", "--count", "1")
		require.NoError(t, err)

		// Genesis is height 0, first lottery is height 1.
		out, err := runCmd(t, "lottery", "verify", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Block Height: #1")
		assert.Contains(t, out, "Verified")
	})
}

func TestLotteryVerify_ByID_Startswith(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "verify-seed-v2", "--count", "1")
		require.NoError(t, err)

		// Grab the record id from history, verify by its full id.
		out, _ := runCmd(t, "lottery", "history")
		id := extractLotteryID(t, out)
		require.NotEmpty(t, id, "expected a lottery id in history output")

		// Exact id match path.
		vout, err := runCmd(t, "lottery", "verify", id)
		require.NoError(t, err)
		assert.Contains(t, vout, "Block Height: #")

		// Substring prefix match fallback: first 8 chars of the id.
		// NOTE: only asserted when the prefix is NOT all-decimal-digits —
		// an all-numeric prefix (e.g. "12345678") is indistinguishable from
		// a block height and is correctly routed to the height lookup, so
		// the substring fallback doesn't apply (covered deterministically by
		// TestLotteryVerify_NumericPrefixID / the height path below). Skip
		// the substring assertion in that ~2% (10/16)^8 case rather than
		// flake.
		prefix := id[:min(8, len(id))]
		if _, perr := strconv.ParseInt(prefix, 10, 64); perr != nil {
			vout, err = runCmd(t, "lottery", "verify", prefix)
			require.NoError(t, err)
			assert.Contains(t, vout, "Block Height: #")
		}

		// Unknown id: not found.
		_, err = runCmd(t, "lottery", "verify", "no-such-lottery")
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "not found")
	})
}

// TestLotteryVerify_NumericPrefixHeight pins the other half of the
// ID-vs-height ambiguity: an all-decimal input (e.g. a numeric id prefix or
// a plain height) is looked up as a height, and a bogus height is "not
// found" rather than crashing or silently falling back to substring search.
func TestLotteryVerify_NumericPrefixIsHeight(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "verify", "12345678")
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "not found")
	})
}

// TestLotteryVerify_NumericPrefixID is a regression test for the
// height-vs-ID disambiguation bug: fmt.Sscanf(input, "%d") accepted a
// PARTIAL numeric prefix, so a record ID that begins with a decimal digit
// (the 16-hex-char hash from CreateLotteryRecord can start with any hex
// char, 0-9 or a-f) was misread as a block height and reported as
// "lottery not found" even though the draw existed.
//
// We plant a record with a digit-prefixed ID directly in the DB to make
// the test deterministic (the CLI has no "choose the id" flag), then
// verify it is found by its exact ID and by an 8-char prefix.
func TestLotteryVerify_NumericPrefixID(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// Establish the DB (tables + a real draw) through the CLI first.
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "numeric-seed", "--count", "1")
		require.NoError(t, err)

		// Rewrite the record's ID to one that begins with a decimal digit,
		// mimicking a 16-hex hash that happens to start with 0-9.
		db := openTestAuroraDB(t)
		_, err = db.Exec(`UPDATE lottery_records SET id = '07551e0cac54a4cf' WHERE id != ''`)
		require.NoError(t, err, "plant the digit-prefixed id")

		// Exact id match must work despite the numeric prefix.
		vout, err := runCmd(t, "lottery", "verify", "07551e0cac54a4cf")
		require.NoError(t, err, "verify by digit-prefixed id should succeed")
		assert.Contains(t, vout, "Block Height: #")

		// Prefix fallback: first 8 chars (also numeric).
		vout, err = runCmd(t, "lottery", "verify", "07551e0c")
		require.NoError(t, err)
		assert.Contains(t, vout, "Block Height: #")
	})
}

// extractLotteryID pulls the id from `lottery history` output. History
// prints each record via record.ToJSON() (json.Marshal) so the output line
// is `{"id":"<uuid>",...}`. The record id appears directly after `"id":"`.
func extractLotteryID(t *testing.T, out string) string {
	t.Helper()
	start := strings.Index(out, `"id":"`)
	if start < 0 {
		t.Logf("no id field found in history output; output=%q", out)
		return ""
	}
	after := out[start+len(`"id":"`):]
	if end := strings.IndexByte(after, '"'); end >= 0 {
		return after[:end]
	}
	return ""
}

func TestLotteryVerify_NotFound(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "verify", "42")
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "not found")
	})
}

func TestLotteryVerify_CorruptedRecord(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "corrupt-me", "--count", "1")
		require.NoError(t, err)

		// Corrupt the stored VRF output so integrity verification fails.
		out, err := runCmd(t, "lottery", "history")
		require.NoError(t, err)
		_ = out

		// Locate the DB file and rewrite the VRF output to an invalid hex
		// string. Simplest deterministic corruption: open the sqlite db and
		// update the record. We reach into the DB directly because the CLI
		// has no "corrupt a record" command.
		corruptLotteryRecord(t)

		_, err = runCmd(t, "lottery", "verify", "1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "integrity check")
	})
}

// corruptLotteryRecord rewrites every lottery record's VRF output to
// invalid hex ("zz") so verify's hex-decode branch trips.
func corruptLotteryRecord(t *testing.T) {
	t.Helper()
	db := openTestAuroraDB(t)

	if _, err := db.Exec(`UPDATE lottery_records SET vrf_output = 'zz'`); err != nil {
		t.Fatalf("corrupt records: %v", err)
	}
}

func TestLotteryExport_Import(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "export-seed", "--count", "1")
		require.NoError(t, err)

		exportPath := filepath.Join(t.TempDir(), "export.json")
		out, err := runCmd(t, "lottery", "export", exportPath)
		require.NoError(t, err)
		assert.Contains(t, out, "Exported 1 lottery records")

		data, err := os.ReadFile(exportPath)
		require.NoError(t, err)
		var records []domainlottery.LotteryRecord
		require.NoError(t, json.Unmarshal(data, &records))
		require.Len(t, records, 1)

		// Re-import into a fresh DB by running against a new temp cwd.
		t.Run("import_to_fresh_db", func(t *testing.T) {
			withTempDir(t, func(t *testing.T) {
				out, err := runCmd(t, "lottery", "import", exportPath)
				require.NoError(t, err)
				assert.Contains(t, out, "Imported 1 lottery records")

				out, err = runCmd(t, "lottery", "history")
				require.NoError(t, err)
				assert.Contains(t, out, "Total lotteries: 1")
			})
		})
	})
}

func TestLotteryImport_InvalidFile(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "bad.json")
		require.NoError(t, os.WriteFile(bad, []byte(`{not json`), 0644))

		_, err := runCmd(t, "lottery", "import", bad)
		require.Error(t, err)

		// Missing file.
		_, err = runCmd(t, "lottery", "import", filepath.Join(t.TempDir(), "nope.json"))
		require.Error(t, err)
	})
}

func TestLotteryImport_PartialFailure(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// One valid record + one that fails validation (empty participants).
		mixed := filepath.Join(t.TempDir(), "mixed.json")
		valid := domainlottery.LotteryRecord{
			ID:              "fixed-valid-id",
			Seed:            "mix-seed-long",
			Participants:    []string{"A", "B"},
			Winners:         []string{"A"},
			WinnerAddresses: []string{"addr"},
			BlockHeight:     7,
			VRFOutput:       strings.Repeat("ab", 32),
			VRFProof:        strings.Repeat("cd", 32),
		}
		invalid := domainlottery.LotteryRecord{ID: "bad", Seed: "x"} // zero participants/winners
		payload, _ := json.Marshal([]domainlottery.LotteryRecord{valid, invalid})
		require.NoError(t, os.WriteFile(mixed, payload, 0644))

		_, err := runCmd(t, "lottery", "import", mixed)
		require.Error(t, err)
	})
}

// TestLotteryImport_RefusesToOverwriteExistingDraw locks the v1.56 audit-trail
// guard: importing a record whose ID already exists must NOT clobber the
// stored draw. The record ID is deterministic (sha256 of seed + VRF output)
// and Save uses INSERT OR REPLACE, so without an existence check a re-import
// or a colliding import would silently replace the stored Verified flag and
// history (TASK-069, ISS-061). We verify by importing the same export into
// the SAME database and asserting neither the record count nor the stored
// draw changes.
func TestLotteryImport_RefusesToOverwriteExistingDraw(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "overwrite-seed", "--count", "1")
		require.NoError(t, err)

		exportPath := filepath.Join(t.TempDir(), "export.json")
		out, err := runCmd(t, "lottery", "export", exportPath)
		require.NoError(t, err)
		assert.Contains(t, out, "Exported 1 lottery records")

		// Snapshot the seed of the stored draw before re-import.
		historyBefore, err := runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, historyBefore, "overwrite-seed")

		// Re-import the same export into the SAME database. The draw already
		// exists, so it must be refused rather than re-saved. Because the only
		// record collides, the import reports a partial failure.
		out, err = runCmd(t, "lottery", "import", exportPath)
		require.Error(t, err, "import of an existing draw should be refused")
		assert.Contains(t, out, "Imported 0 of 1")

		// The stored audit trail is unchanged: still one record, same seed.
		historyAfter, err := runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, historyAfter, "overwrite-seed")
		assert.Contains(t, historyAfter, "Total lotteries: 1")
	})
}

// TestLotteryImport_AllowsSameIDWithNoExistingRecord ensures the existence
// check does not regress the legitimate fresh-import path (a valid record
// whose ID does not yet exist must still be imported successfully).
func TestLotteryImport_AllowsFreshValidImport(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		importPath := filepath.Join(t.TempDir(), "fresh.json")
		rec := domainlottery.LotteryRecord{
			ID:              "fresh-import-id",
			Seed:            "fresh-import-seed-long",
			Participants:    []string{"A", "B"},
			Winners:         []string{"A"},
			WinnerAddresses: []string{"addr"},
			BlockHeight:     3,
			VRFOutput:       strings.Repeat("ef", 32),
			VRFProof:        strings.Repeat("12", 32),
		}
		payload, _ := json.Marshal([]domainlottery.LotteryRecord{rec})
		require.NoError(t, os.WriteFile(importPath, payload, 0644))

		out, err := runCmd(t, "lottery", "import", importPath)
		require.NoError(t, err)
		assert.Contains(t, out, "Imported 1 lottery records")
	})
}

func TestLotteryStats(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B,C", "--seed", "stats-seed", "--count", "1")
		require.NoError(t, err)

		out, err := runCmd(t, "lottery", "stats")
		require.NoError(t, err)
		assert.Contains(t, out, "Total lotteries: 1")
	})
}

// TestLotteryStats_AgreesWithHistoryForImportedRecord locks the v1.57 fix:
// lottery stats must count the persistent lottery_records store (the same
// source as history/verify/export/import), not the in-memory chain. Imported
// draws only write lottery_records and never reach the chain, so before the
// fix stats reported 0 for an imported record that history correctly showed.
func TestLotteryStats_AgreesWithHistoryForImportedRecord(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		importPath := filepath.Join(t.TempDir(), "imp.json")
		rec := domainlottery.LotteryRecord{
			ID:              "stats-import-id",
			Seed:            "stats-import-seed-long",
			Participants:    []string{"A", "B"},
			Winners:         []string{"A"},
			WinnerAddresses: []string{"addr"},
			BlockHeight:     5,
			VRFOutput:       strings.Repeat("ef", 32),
			VRFProof:        strings.Repeat("12", 32),
		}
		payload, _ := json.Marshal([]domainlottery.LotteryRecord{rec})
		require.NoError(t, os.WriteFile(importPath, payload, 0644))

		out, err := runCmd(t, "lottery", "import", importPath)
		require.NoError(t, err)
		require.Contains(t, out, "Imported 1 lottery records")

		hist, err := runCmd(t, "lottery", "history")
		require.NoError(t, err)
		require.Contains(t, hist, "Total lotteries: 1")

		stats, err := runCmd(t, "lottery", "stats")
		require.NoError(t, err)
		require.Contains(t, stats, "Total lotteries: 1",
			"stats must agree with history (persistent lottery_records), not the in-memory chain")
		require.Contains(t, stats, "Latest block: #5")
	})
}

func TestLotteryReset_NeedsConfirmation(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "reset-seed", "--count", "1")
		require.NoError(t, err)

		// Without --yes: warns and does nothing.
		out, err := runCmd(t, "lottery", "reset")
		require.NoError(t, err)
		assert.Contains(t, out, "delete ALL lottery records")
	})
}

func TestLotteryReset_WithYes(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "reset-seed-2", "--count", "1")
		require.NoError(t, err)

		out, err := runCmd(t, "lottery", "reset", "--yes")
		require.NoError(t, err)
		assert.Contains(t, out, "reset complete")

		// Reset must clear BOTH the chain blocks and the persistent
		// lottery_records (the warning says "delete ALL lottery records!"),
		// so history after a reset shows nothing.
		out, err = runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, out, "No lottery records found")
	})
}

func TestLotteryReset_OnFreshDB(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// A brand-new DB has no lottery_records table yet; reset must
		// tolerate the "no such table" error instead of failing.
		out, err := runCmd(t, "lottery", "reset", "--yes")
		require.NoError(t, err)
		assert.Contains(t, out, "reset complete")
	})
}

func TestLotteryDBInfo(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		out, err := runCmd(t, "lottery", "db-info")
		require.NoError(t, err)
		assert.Contains(t, out, "Database Info")
		assert.Contains(t, out, "Total blocks: 0")
	})
}
