package lotteryapp

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

type emptyLotteryRepo struct{}

func (emptyLotteryRepo) Save(*lottery.LotteryRecord) error { return nil }

func (emptyLotteryRepo) GetByID(string) (*lottery.LotteryRecord, error) {
	return nil, lottery.ErrNotFound
}

func (emptyLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) { return nil, nil }

func (emptyLotteryRepo) Count() (int, error) { return 0, nil }

func (emptyLotteryRepo) List(int, int) ([]*lottery.LotteryRecord, error) { return nil, nil }
func (emptyLotteryRepo) ListMatching(string, int, int) ([]*lottery.LotteryRecord, error) {
	return nil, nil
}

func (emptyLotteryRepo) GetByBlockHeight(int64) ([]*lottery.LotteryRecord, error) {
	return nil, nil
}

// TestExportRecords_EmptyIsAnArray pins the portable format's empty case. A
// nil repository slice used to marshal as `null`, so an empty database
// exported a document that was not the array `aurora lottery import` reads.
func TestExportRecords_EmptyIsAnArray(t *testing.T) {
	document, count, err := ExportRecords(emptyLotteryRepo{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.JSONEq(t, `[]`, string(document), "an empty export must be an empty JSON array, not null")
}

// TestImportRecords_RejectsNull pins the other half: json.Unmarshal accepts a
// literal null into a slice, so the CLI and desktop used to report a
// successful import of 0 records for a file that is not an export.
func TestImportRecords_RejectsNull(t *testing.T) {
	repo := &mockLotteryRepo{}

	result, err := ImportRecords(repo, []byte("null"))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidImportDocument)
	assert.Equal(t, "parse document: expected a JSON array of lottery records, got null", err.Error())
	assert.Empty(t, repo.records, "a null document must store nothing")
	assert.Zero(t, result.Imported)
}

func TestImportRecords_RejectsMalformedJSONWithTypedError(t *testing.T) {
	repo := &mockLotteryRepo{}

	_, err := ImportRecords(repo, []byte(`{not json`))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidImportDocument)
	assert.Contains(t, err.Error(), "parse document")
	assert.Empty(t, repo.records)
}

func genuinePortableRecord(t *testing.T) lottery.LotteryRecord {
	t.Helper()
	winners, addresses, output, proof, publicKey, err := lottery.NewService().DrawWinners(
		[]string{"Alice", "Bob", "Carol"}, "portable-integrity-seed", 2,
	)
	require.NoError(t, err)
	record := lottery.CreateLotteryRecord(
		"portable-integrity-seed",
		[]string{"Alice", "Bob", "Carol"},
		winners,
		addresses,
		output,
		proof,
		publicKey,
		7,
	)
	return *record
}

func TestImportRecords_DerivesVerifiedFromStoredProof(t *testing.T) {
	repo := &mockLotteryRepo{}
	record := genuinePortableRecord(t)
	record.Verified = true
	document, err := json.Marshal([]lottery.LotteryRecord{record})
	require.NoError(t, err)

	result, err := ImportRecords(repo, document)
	require.NoError(t, err)
	assert.Equal(t, ImportResult{Total: 1, Imported: 1}, result)
	require.Len(t, repo.records, 1)
	assert.NotEmpty(t, repo.records[0].VRFPublicKey)
	assert.True(t, repo.records[0].Verified, "a valid keyed proof must be marked verified")
}

func TestImportRecords_DoesNotTrustVerifiedWithoutAKey(t *testing.T) {
	repo := &mockLotteryRepo{}
	record := genuinePortableRecord(t)
	record.VRFPublicKey = ""
	record.Verified = true
	document, err := json.Marshal([]lottery.LotteryRecord{record})
	require.NoError(t, err)

	result, err := ImportRecords(repo, document)
	require.NoError(t, err)
	assert.Equal(t, ImportResult{Total: 1, Imported: 1}, result)
	require.Len(t, repo.records, 1)
	assert.False(t, repo.records[0].Verified, "keyless legacy data cannot claim cryptographic verification")
}

func TestImportRecords_RejectsAnInvalidKeyedProof(t *testing.T) {
	repo := &mockLotteryRepo{}
	record := genuinePortableRecord(t)
	record.VRFProof = ""
	record.Verified = true
	document, err := json.Marshal([]lottery.LotteryRecord{record})
	require.NoError(t, err)

	result, err := ImportRecords(repo, document)
	require.NoError(t, err)
	assert.Equal(t, ImportResult{Total: 1, Rejected: []int{0}}, result)
	assert.Empty(t, repo.records, "an invalid proof must not be persisted")
}

func TestImportRecords_RejectsATamperedWinnerAddress(t *testing.T) {
	repo := &mockLotteryRepo{}
	record := genuinePortableRecord(t)
	record.WinnerAddresses[0] = lottery.NameToAddress("Mallory")
	document, err := json.Marshal([]lottery.LotteryRecord{record})
	require.NoError(t, err)

	result, err := ImportRecords(repo, document)
	require.NoError(t, err)
	assert.Equal(t, ImportResult{Total: 1, Rejected: []int{0}}, result)
	assert.Empty(t, repo.records, "a tampered payout address must not be imported")
}

func TestImportRecords_RejectsRecordIDNotDerivedFromEvidence(t *testing.T) {
	repo := &mockLotteryRepo{}
	record := genuinePortableRecord(t)
	record.ID = "0000000000000000"
	document, err := json.Marshal([]lottery.LotteryRecord{record})
	require.NoError(t, err)

	result, err := ImportRecords(repo, document)
	require.NoError(t, err)
	assert.Equal(t, ImportResult{Total: 1, Rejected: []int{0}}, result)
	assert.Empty(t, repo.records, "a forged ID must not bypass duplicate-draw detection")
}

type pathLotteryRepo struct {
	emptyLotteryRepo
	path string
}

func (r pathLotteryRepo) DBPath() string { return r.path }

// TestWriteRecordsToFile_RefusesTheLiveDatabase covers the shared export entry
// point: the file it would replace may be the database itself, an alias of it,
// or one of SQLite's sidecars.
func TestWriteRecordsToFile_RefusesTheLiveDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("not really sqlite"), 0o644))
	repo := pathLotteryRepo{path: dbPath}

	for _, dest := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
		_, err := WriteRecordsToFile(repo, dest)
		require.Error(t, err, "exporting to %s must be refused", dest)
		assert.Contains(t, err.Error(), "live database")
	}

	data, err := os.ReadFile(dbPath)
	require.NoError(t, err)
	assert.Equal(t, "not really sqlite", string(data), "a refused export must leave the database alone")
}

func TestWriteRecordsToFile_RefusesDiskFileURI(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state with spaces", "aurora.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
		require.NoError(t, os.WriteFile(path, []byte("live state"), 0o600))
	}
	dsn := "file:" + url.PathEscape(filepath.ToSlash(dbPath))
	repo := pathLotteryRepo{path: dsn}

	for _, dest := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
		_, err := WriteRecordsToFile(repo, dest)
		require.Error(t, err, "exporting to %s through a file URI must be refused", dest)
		assert.Contains(t, err.Error(), "live database")

		data, readErr := os.ReadFile(dest)
		require.NoError(t, readErr)
		assert.Equal(t, "live state", string(data), "a refused export must leave the live file alone")
	}
}

func TestWriteRecordsToFile_RefusesASymlinkAlias(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	alias := filepath.Join(dir, "draws.json")
	require.NoError(t, os.WriteFile(dbPath, []byte("not really sqlite"), 0o644))
	require.NoError(t, os.Symlink(dbPath, alias))

	_, err := WriteRecordsToFile(pathLotteryRepo{path: dbPath}, alias)
	require.Error(t, err, "a symlink to the database is the database")
	assert.Contains(t, err.Error(), "live database")
}

func TestWriteRecordsToFile_RefusesSidecarsOfSymlinkedDatabase(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	linkDir := filepath.Join(root, "link")
	require.NoError(t, os.MkdirAll(realDir, 0o755))
	require.NoError(t, os.MkdirAll(linkDir, 0o755))

	dbPath := filepath.Join(realDir, "aurora.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("not really sqlite"), 0o600))
	linkPath := filepath.Join(linkDir, "aurora.db")
	if err := os.Symlink(dbPath, linkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	repo := pathLotteryRepo{path: linkPath}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		dest := dbPath + suffix
		require.NoError(t, os.WriteFile(dest, []byte("live sidecar"), 0o600))

		_, err := WriteRecordsToFile(repo, dest)
		require.Error(t, err, "exporting over the real sidecar %s must be refused", suffix)
		assert.Contains(t, err.Error(), "live database")

		data, readErr := os.ReadFile(dest)
		require.NoError(t, readErr)
		assert.Equal(t, "live sidecar", string(data))
	}
}

func TestWriteRecordsToFile_WritesAnEmptyArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	count, err := WriteRecordsToFile(emptyLotteryRepo{}, path)
	require.NoError(t, err)
	assert.Zero(t, count)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, "[]", string(data))
}

// TestReadImportFileBoundsTheDocument: an oversized file is refused before the
// parser sees it, through a sentinel a presentation layer can name instead of
// echoing a byte count.
func TestReadImportFileBoundsTheDocument(t *testing.T) {
	dir := t.TempDir()

	small := filepath.Join(dir, "draws.json")
	require.NoError(t, os.WriteFile(small, []byte(`{"draws":[]}`), 0o644))
	data, err := ReadImportFile(small)
	require.NoError(t, err)
	require.JSONEq(t, `{"draws":[]}`, string(data))

	// A sparse file past the limit: the reader stops at MaxImportFileSize+1
	// bytes, so the test does not need to write 64 MiB of real content.
	big := filepath.Join(dir, "big.json")
	f, err := os.Create(big)
	require.NoError(t, err)
	require.NoError(t, f.Truncate(MaxImportFileSize+1))
	require.NoError(t, f.Close())

	_, err = ReadImportFile(big)
	require.ErrorIs(t, err, ErrImportFileTooLarge)

	_, err = ReadImportFile(filepath.Join(dir, "missing.json"))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrImportFileTooLarge)
}
