package deskapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
	domainlottery "github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

func genuineDesktopLotteryRecord(t *testing.T, seed string) domainlottery.LotteryRecord {
	t.Helper()
	participants := []string{"Alice", "Bob", "Carol"}
	winners, addresses, output, proof, publicKey, err := domainlottery.NewService().DrawWinners(
		participants, seed, 2,
	)
	require.NoError(t, err)
	record := domainlottery.CreateLotteryRecord(
		seed, participants, winners, addresses, output, proof, publicKey, 1,
	)
	record.Verified = true
	return *record
}

// TestExportImport_RoundTripThroughSharedAppLayer pins the contract the desktop
// menu actions rely on: the SAME app-layer functions the CLI calls move a draw
// out of one database and into another, and the overwrite guard still refuses a
// re-import.
//
// The menu handlers add only the file dialogs around these calls, so this test
// covers the behavior a desktop Export/Import would produce without opening a
// window.
func TestExportImport_RoundTripThroughSharedAppLayer(t *testing.T) {
	src, err := sqlite.NewLotteryRepository(filepath.Join(t.TempDir(), "source.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = src.Close() })

	record := genuineDesktopLotteryRecord(t, "desktop-roundtrip-seed")
	record.BlockHeight = 11
	require.NoError(t, src.Save(&record))

	document, count, err := lotteryapp.ExportRecords(src)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Contains(t, string(document), `"`+record.ID+`"`)

	dst, err := sqlite.NewLotteryRepository(filepath.Join(t.TempDir(), "dest.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = dst.Close() })

	result, err := lotteryapp.ImportRecords(dst, document)
	require.NoError(t, err)
	require.Equal(t, lotteryapp.ImportResult{Total: 1, Imported: 1}, result)

	got, err := dst.GetByID(record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.Winners, got.Winners)
	assert.Equal(t, record.VRFPublicKey, got.VRFPublicKey, "the proof key must survive export and import")
	assert.True(t, got.Verified, "the Verified flag must survive the round trip")

	// Re-importing into the same database must be refused, not silently
	// replace the stored draw (TASK-069, ISS-061) - the desktop menu reports
	// this as a warning, exactly like the CLI's non-zero "partial failure".
	again, err := lotteryapp.ImportRecords(dst, document)
	require.NoError(t, err)
	assert.Equal(t, lotteryapp.ImportResult{Total: 1, Imported: 0, Rejected: []int{0}}, again)
}

// TestImportRecords_RejectsMalformedDocument keeps a corrupt file from being
// half-applied: the document must parse before anything is written.
func TestImportRecords_RejectsMalformedDocument(t *testing.T) {
	repo, err := sqlite.NewLotteryRepository(filepath.Join(t.TempDir(), "bad.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	_, err = lotteryapp.ImportRecords(repo, []byte(`{not json`))
	require.Error(t, err)

	records, err := repo.GetAll()
	require.NoError(t, err)
	assert.Empty(t, records, "a malformed document must store nothing")
}

func TestImportRecords_RejectsNullDocument(t *testing.T) {
	repo, err := sqlite.NewLotteryRepository(filepath.Join(t.TempDir(), "null.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	_, err = lotteryapp.ImportRecords(repo, []byte("null"))
	require.Error(t, err, "null is not a portable lottery document")

	records, err := repo.GetAll()
	require.NoError(t, err)
	assert.Empty(t, records)
}

// TestImportRecords_PartialFailureMatchesCLIContract covers the mixed file a
// user is most likely to produce: one valid draw and one that fails
// validation. The valid one lands and the invalid one is reported by index.
func TestImportRecords_PartialFailureMatchesCLIContract(t *testing.T) {
	repo, err := sqlite.NewLotteryRepository(filepath.Join(t.TempDir(), "mixed.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	valid := genuineDesktopLotteryRecord(t, "mixed-seed-long")
	invalid := domainlottery.LotteryRecord{ID: "bad", Seed: "x"} // no participants/winners

	document, err := json.Marshal([]domainlottery.LotteryRecord{valid, invalid})
	require.NoError(t, err)

	result, err := lotteryapp.ImportRecords(repo, document)
	require.NoError(t, err)
	assert.Equal(t, lotteryapp.ImportResult{Total: 2, Imported: 1, Rejected: []int{1}}, result)
}

// TestExportLotteriesReplacesTheDocumentAtomically pins the menu path, not
// just the shared writer: File > Export must produce the same document the CLI
// does, keep an existing file's permissions, and leave no temporary sibling.
func TestExportLotteriesReplacesTheDocumentAtomically(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	record := domainlottery.LotteryRecord{
		ID:              "menu-export-id",
		BlockHeight:     3,
		Seed:            "menu-export-seed",
		Participants:    []string{"A", "B"},
		Winners:         []string{"A"},
		WinnerAddresses: []string{"addr"},
		VRFOutput:       "0011223344556677889900aabbccddeeff00112233445566778899aabbccddee",
		VRFProof:        "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100",
		Verified:        true,
	}
	require.NoError(t, repo.Save(&record))
	require.NoError(t, repo.Close())

	exportPath := filepath.Join(t.TempDir(), "aurora-lotteries.json")
	require.NoError(t, os.WriteFile(exportPath, []byte(`["stale"]`), 0o600))
	require.NoError(t, os.Chmod(exportPath, 0o640))

	originalSave := saveDialog
	originalBox := messageBox
	originalReady := appReady
	t.Cleanup(func() {
		saveDialog = originalSave
		messageBox = originalBox
		appReady = originalReady
	})
	saveDialog = func(mygo.SaveDialogOptions) (string, error) { return exportPath, nil }
	var dialogs []mygo.MessageOptions
	messageBox = func(opts mygo.MessageOptions) { dialogs = append(dialogs, opts) }
	appReady = func() bool { return false }

	exportLotteries(About{DBPath: dbPath}, nil)

	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "menu-export-id")
	assert.NotContains(t, string(data), `"stale"`)

	info, err := os.Stat(exportPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())

	entries, err := os.ReadDir(filepath.Dir(exportPath))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "the export must not leave a temporary sibling")

	require.Len(t, dialogs, 1)
	assert.Equal(t, mygo.MessageInfo, dialogs[0].Type)
	assert.Equal(t, "Lotteries exported", dialogs[0].Title)
}

// TestExportLotteriesRefusesTheLiveDatabase pins the menu half of the guard:
// choosing the data directory in the save dialog must not replace aurora.db
// with the JSON export, and the failure must reach the user.
func TestExportLotteriesRefusesTheLiveDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	require.NoError(t, repo.Close())

	before, err := os.ReadFile(dbPath)
	require.NoError(t, err)

	originalSave := saveDialog
	originalBox := messageBox
	t.Cleanup(func() { saveDialog = originalSave; messageBox = originalBox })
	saveDialog = func(mygo.SaveDialogOptions) (string, error) { return dbPath, nil }
	var dialogs []mygo.MessageOptions
	messageBox = func(opts mygo.MessageOptions) { dialogs = append(dialogs, opts) }

	exportLotteries(About{DBPath: dbPath}, nil)

	require.Len(t, dialogs, 1)
	assert.Equal(t, mygo.MessageError, dialogs[0].Type)
	assert.Equal(t, "Export failed", dialogs[0].Title)

	after, err := os.ReadFile(dbPath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the menu must not overwrite the live database")

	info, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err, "the database must still open as SQLite")
	require.NoError(t, info.Close())
}
