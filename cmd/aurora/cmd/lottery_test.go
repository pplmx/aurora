package cmd

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestLotteryCreate_DefaultCount regression (TASK-108, ISS-100): the --count
// flag was registered with viper.GetInt("lottery.defaultCount") at init() time —
// before initConfig/setDefaultConfig load — so its default was always 0 and any
// create without -c failed with "winner count must be positive". It must now
// fall back to the configured default (3) at run time.
func TestLotteryCreate_DefaultCount(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		out, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob,Charlie,Dave,Eve", "--seed", "default-count-seed")
		require.NoError(t, err, "lottery create without --count must succeed (default 3)")
		assert.Contains(t, out, "Winners:")
		// The default of 3 must actually draw 3 winners from the 5 participants.
		winnerLine := regexp.MustCompile(`^\s+\d+\. `)
		winnerLines := 0
		for _, line := range strings.Split(out, "\n") {
			if winnerLine.MatchString(line) {
				winnerLines++
			}
		}
		assert.Equal(t, 3, winnerLines, "expected 3 drawn winners by default")
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

func TestLotteryHistory_Pages(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		for _, seed := range []string{"page-seed-one", "page-seed-two", "page-seed-three"} {
			_, err := runCmd(t, "lottery", "create",
				"--participants", "Alice,Bob", "--seed", seed, "--count", "1")
			require.NoError(t, err)
		}

		out, err := runCmd(t, "lottery", "history", "--limit", "1", "--offset", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Lotteries on this page: 1 (offset 1)")
		assert.Contains(t, out, "page-seed-two")
		assert.NotContains(t, out, "page-seed-three")
		assert.NotContains(t, out, "page-seed-one")

		out, err = runCmd(t, "lottery", "history", "--limit", "1", "--offset", "99")
		require.NoError(t, err)
		assert.Contains(t, out, "Lotteries on this page: 0 (offset 99)")
		assert.NotContains(t, out, "No lottery records found",
			"an empty later page must not claim the whole store is empty")

		_, err = runCmd(t, "lottery", "history", "--limit", "-1")
		require.Error(t, err)
		_, err = runCmd(t, "lottery", "history", "--offset", "-1")
		require.Error(t, err)
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
		assert.Contains(t, out, "VRF Public Key:", "SQLite must preserve the proof key for cryptographic verification")
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
// We plant a canonical keyless draw whose deterministic ID starts with a
// decimal digit (the CLI has no "choose the id" flag), then verify it is found
// by its exact ID and by an 8-char prefix. The seed/output pair is fixed so
// the ID remains derivable and passes the integrity verifier.
func TestLotteryVerify_NumericPrefixID(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// Establish the DB (tables + a real draw) through the CLI first.
		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "numeric-seed", "--count", "1")
		require.NoError(t, err)

		seed := "numeric-seed"
		participants := []string{"Alice", "Bob"}
		output := []byte{0x00}
		winners := lottery.SelectWinners(output, participants, 1)
		record := lottery.CreateLotteryRecord(
			seed,
			participants,
			winners,
			[]string{lottery.NameToAddress(winners[0])},
			output,
			[]byte{0x00},
			"",
			1,
		)
		const numericID = "3f6b943059e93f5f"
		require.Equal(t, numericID, record.ID)

		participantsJSON, err := json.Marshal(record.Participants)
		require.NoError(t, err)
		winnersJSON, err := json.Marshal(record.Winners)
		require.NoError(t, err)
		addressesJSON, err := json.Marshal(record.WinnerAddresses)
		require.NoError(t, err)

		// Replace the random draw with the fixed canonical record. It has no
		// stored VRF key, exercising the legacy deterministic-winner path.
		db := openTestAuroraDB(t)
		_, err = db.Exec(`DELETE FROM lottery_vrf_keys`)
		require.NoError(t, err)
		_, err = db.Exec(`
			UPDATE lottery_records
			SET id = ?, seed = ?, participants = ?, winners = ?,
			    winner_addresses = ?, vrf_proof = ?, vrf_output = ?, verified = 0
			WHERE id != ''`,
			record.ID, record.Seed, string(participantsJSON), string(winnersJSON),
			string(addressesJSON), record.VRFProof, record.VRFOutput,
		)
		require.NoError(t, err, "plant the canonical digit-prefixed draw")

		// Exact id match must work despite the numeric prefix.
		vout, err := runCmd(t, "lottery", "verify", numericID)
		require.NoError(t, err, "verify by digit-prefixed id should succeed")
		assert.Contains(t, vout, "Block Height: #")

		// Prefix fallback: first 8 chars (also numeric).
		vout, err = runCmd(t, "lottery", "verify", numericID[:8])
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

func genuineLotteryRecord(t *testing.T, seed string, blockHeight int64) lottery.LotteryRecord {
	t.Helper()
	participants := []string{"Alice", "Bob", "Carol"}
	winners, addresses, output, proof, publicKey, err := lottery.NewService().DrawWinners(
		participants, seed, 2,
	)
	require.NoError(t, err)
	record := lottery.CreateLotteryRecord(
		seed, participants, winners, addresses, output, proof, publicKey, blockHeight,
	)
	record.Verified = true
	return *record
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
		var records []lottery.LotteryRecord
		require.NoError(t, json.Unmarshal(data, &records))
		require.Len(t, records, 1)
		require.NotEmpty(t, records[0].VRFPublicKey, "export must carry the persisted proof key")
		id := records[0].ID

		// Re-import into a fresh DB by running against a new temp cwd.
		t.Run("import_to_fresh_db", func(t *testing.T) {
			withTempDir(t, func(t *testing.T) {
				out, err := runCmd(t, "lottery", "import", exportPath)
				require.NoError(t, err)
				assert.Contains(t, out, "Imported 1 lottery records")

				out, err = runCmd(t, "lottery", "history")
				require.NoError(t, err)
				assert.Contains(t, out, "Total lotteries: 1")

				out, err = runCmd(t, "lottery", "verify", id)
				require.NoError(t, err)
				assert.Contains(t, out, "VRF Public Key:")
				assert.NotContains(t, out, "pre-v1.31",
					"an imported draw with its key must retain cryptographic verification")
			})
		})
	})
}

// TestLotteryExport_EmptyDatabaseIsAnArray pins the document an empty database
// produces. The nil slice used to marshal as `null`, which the importer then
// accepted as "0 records imported" even though it is not an export document.
func TestLotteryExport_EmptyDatabaseIsAnArray(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		exportPath := filepath.Join(t.TempDir(), "empty.json")
		out, err := runCmd(t, "lottery", "export", exportPath)
		require.NoError(t, err)
		assert.Contains(t, out, "Exported 0 lottery records")

		data, err := os.ReadFile(exportPath)
		require.NoError(t, err)
		assert.JSONEq(t, `[]`, string(data), "an empty export must be a JSON array")
	})
}

// TestLotteryExport_RefusesToOverwriteTheDatabase pins the guard the probe
// found missing: the export used to replace the SQLite file with JSON and
// print success, and only the already-open handle kept history working.
func TestLotteryExport_RefusesToOverwriteTheDatabase(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "self-export", "--count", "1")
		require.NoError(t, err)

		dbPath := "./data/aurora.db"
		before, err := os.ReadFile(dbPath)
		require.NoError(t, err)
		require.Equal(t, "SQLite format 3\x00", string(before[:16]))

		_, err = runCmd(t, "lottery", "export", dbPath)
		require.Error(t, err, "exporting over the live database must be refused")
		assert.Contains(t, err.Error(), "live database")

		after, err := os.ReadFile(dbPath)
		require.NoError(t, err)
		assert.Equal(t, before, after, "a refused export must not touch the database")
	})
}

func TestLotteryImport_RejectsNullDocument(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "null.json")
		require.NoError(t, os.WriteFile(path, []byte("null"), 0o644))

		_, err := runCmd(t, "lottery", "import", path)
		require.Error(t, err, "null is not a portable export and must not report a successful import")
		assert.Contains(t, err.Error(), "null")
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
		valid := genuineLotteryRecord(t, "mix-seed-long", 7)
		invalid := lottery.LotteryRecord{ID: "bad", Seed: "x"} // zero participants/winners
		payload, _ := json.Marshal([]lottery.LotteryRecord{valid, invalid})
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
		rec := genuineLotteryRecord(t, "fresh-import-seed-long", 3)
		payload, _ := json.Marshal([]lottery.LotteryRecord{rec})
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
		rec := genuineLotteryRecord(t, "stats-import-seed-long", 5)
		payload, _ := json.Marshal([]lottery.LotteryRecord{rec})
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

		// Without the gate: refuses loudly (non-zero exit) so a script can tell
		// the destructive reset did NOT run (TASK-100, ISS-092; consistent
		// with the backup restore --confirm contract). The warning text is
		// the error message.
		_, err = runCmd(t, "lottery", "reset")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete ALL lottery records")
	})
}

func TestLotteryReset_WithYesShorthand(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "reset-seed-2", "--count", "1")
		require.NoError(t, err)

		out, err := runCmd(t, "lottery", "reset", "-y")
		require.NoError(t, err)
		assert.Contains(t, out, "reset complete")

		// Reset must clear BOTH the chain blocks and the persistent
		// lottery_records (the warning says "delete ALL lottery records!"),
		// so history after a reset shows nothing.
		out, err = runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, out, "No lottery records found")

		db := openTestAuroraDB(t)
		var keyCount int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM lottery_vrf_keys`).Scan(&keyCount))
		assert.Zero(t, keyCount, "reset must clear the proof-key side table")
	})
}

func TestLotteryReset_WithConfirmFlag(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "reset-seed-c", "--count", "1")
		require.NoError(t, err)

		// The reset takes the shared destructive gate spelling (--confirm).
		out, err := runCmd(t, "lottery", "reset", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "reset complete")
	})
}

func TestLotteryReset_OnFreshDB(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// A brand-new DB has no lottery_records table yet; reset must
		// tolerate the "no such table" error instead of failing.
		out, err := runCmd(t, "lottery", "reset", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "reset complete")
	})
}

// TestLotteryReset_RestartsBlockHeightAfterCreate locks the v1.58 fix: reset
// must re-seed the in-memory chain back to genesis, so a create after a reset
// starts at block height #1. Before the fix the chain singleton kept the
// pre-reset blocks in memory, so a post-reset draw got a stale block height
// (e.g. #3 after two prior draws) and a PrevHash referencing a deleted block
// (TASK-071, ISS-063).
func TestLotteryReset_RestartsBlockHeightAfterCreate(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// Two draws before the reset => chain would otherwise be at height 2.
		for i, seed := range []string{"pre-one", "pre-two"} {
			_, err := runCmd(t, "lottery", "create",
				"--participants", "A,B", "--seed", seed, "--count", "1")
			require.NoError(t, err, "create %d", i)
		}

		out, err := runCmd(t, "lottery", "reset", "--confirm")
		require.NoError(t, err)
		require.Contains(t, out, "reset complete")

		// Create a fresh draw after the reset. Its block height must restart
		// at 1 (a valid genesis-anchored chain), not continue from the stale
		// pre-reset count.
		out, err = runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "post-reset", "--count", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Block height: #1",
			"post-reset draw must restart at block height #1")

		// And the persisted history reflects exactly that one fresh draw.
		out, err = runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, out, "Total lotteries: 1")
	})
}

// TestLotteryInfoCommandsPrintJSON: the summary and the database report are the
// two commands an operator scripts against, so both print one document. The
// latest block is omitted before the first draw rather than reported as zero
// (which is the genesis height).
func TestLotteryInfoCommandsPrintJSON(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		out, err := runCmd(t, "lottery", "stats", "--json")
		require.NoError(t, err)
		var stats struct {
			Total       int    `json:"total"`
			DBPath      string `json:"db_path"`
			LatestBlock *int64 `json:"latest_block"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &stats), "got %q", out)
		assert.Equal(t, 0, stats.Total)
		assert.NotEmpty(t, stats.DBPath)
		assert.Nil(t, stats.LatestBlock, "no draw yet: latest_block is omitted")

		out, err = runCmd(t, "lottery", "db-info", "--json")
		require.NoError(t, err)
		var info struct {
			Path      string `json:"path"`
			SizeBytes int64  `json:"size_bytes"`
			Blocks    int    `json:"blocks"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &info), "got %q", out)
		assert.NotEmpty(t, info.Path)
		assert.Greater(t, info.SizeBytes, int64(0))
		assert.Equal(t, 0, info.Blocks)

		_, err = runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "info-json", "--count", "1")
		require.NoError(t, err)

		out, err = runCmd(t, "lottery", "stats", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &stats))
		assert.Equal(t, 1, stats.Total)
		require.NotNil(t, stats.LatestBlock)
		assert.Equal(t, int64(1), *stats.LatestBlock)
	})
}

// TestLotteryVerifyFailsOnTamperedRecord: the stored winners no longer match
// the seed, so both the table and the JSON mode must exit non-zero while still
// printing what failed.
func TestLotteryVerifyFailsOnTamperedRecord(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		_, err := runCmd(t, "lottery", "create",
			"--participants", "A,B", "--seed", "tamper-seed", "--count", "1")
		require.NoError(t, err)

		var draws []struct {
			ID string `json:"id"`
		}
		out, err := runCmd(t, "lottery", "history", "--json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(out), &draws))
		require.Len(t, draws, 1)

		out, err = runCmd(t, "lottery", "verify", draws[0].ID, "--json")
		require.NoError(t, err)
		assert.Contains(t, out, `"integrity_ok": true`)

		db, err := sql.Open("sqlite", blockchain.DBPath())
		require.NoError(t, err)
		_, err = db.Exec(`UPDATE lottery_records SET winners = '["Mallory"]'`)
		require.NoError(t, err)
		require.NoError(t, db.Close())

		out, err = runCmd(t, "lottery", "verify", draws[0].ID, "--json")
		require.Error(t, err, "a failed check must be a non-zero exit in JSON mode")
		var report struct {
			IntegrityOK bool `json:"integrity_ok"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &report), "the report must still print: %q", out)
		assert.False(t, report.IntegrityOK)

		out, err = runCmd(t, "lottery", "verify", draws[0].ID)
		require.Error(t, err)
		assert.Contains(t, out, "❌")
	})
}

func TestLotteryDBInfo(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		out, err := runCmd(t, "lottery", "db-info")
		require.NoError(t, err)
		assert.Contains(t, out, "Database Info")
		assert.Regexp(t, `Database size: \d+(\.\d+)? (B|KiB|MiB|GiB)`, out)
		assert.Contains(t, out, "Total blocks: 0")
	})
}

// TestLotteryImportRejectsOversizedFile: the reader's 64 MiB cap is a user
// mistake, so the CLI names the limit in the active language instead of
// printing the raw byte count. The file is sparse, so the test does not write
// 64 MiB of content.
func TestLotteryImportRejectsOversizedFile(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		big := filepath.Join(t.TempDir(), "big.json")
		f, err := os.Create(big)
		require.NoError(t, err)
		require.NoError(t, f.Truncate(lotteryapp.MaxImportFileSize+1))
		require.NoError(t, f.Close())

		_, err = runCmd(t, "lottery", "import", big)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "import file is larger than 64.0 MiB")
	})
}
