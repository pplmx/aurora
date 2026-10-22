package lottery

import (
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
