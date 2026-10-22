package lottery

import (
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
	assert.Contains(t, err.Error(), "null")
	assert.Empty(t, repo.records, "a null document must store nothing")
	assert.Zero(t, result.Imported)
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

	for _, dest := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		_, err := WriteRecordsToFile(repo, dest)
		require.Error(t, err, "exporting to %s must be refused", dest)
		assert.Contains(t, err.Error(), "live database")
	}

	data, err := os.ReadFile(dbPath)
	require.NoError(t, err)
	assert.Equal(t, "not really sqlite", string(data), "a refused export must leave the database alone")
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

func TestWriteRecordsToFile_WritesAnEmptyArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	count, err := WriteRecordsToFile(emptyLotteryRepo{}, path)
	require.NoError(t, err)
	assert.Zero(t, count)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, "[]", string(data))
}
