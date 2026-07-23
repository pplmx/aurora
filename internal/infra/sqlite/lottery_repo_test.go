package sqlite

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

func setupLotteryTestDB(t *testing.T) (*LotteryRepository, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_lottery.db")

	repo, err := NewLotteryRepository(dbPath)
	require.NoError(t, err, "NewLotteryRepository should succeed")

	cleanup := func() {
		if repo != nil {
			_ = repo.Close()
		}
		_ = os.RemoveAll(tmpDir)
	}
	return repo, cleanup
}

func sampleLotteryRecord(id string, blockHeight int64) *lottery.LotteryRecord {
	return &lottery.LotteryRecord{
		ID:              id,
		BlockHeight:     blockHeight,
		Seed:            "seed-" + id,
		Participants:    []string{"alice", "bob", "carol"},
		Winners:         []string{"alice", "bob"},
		WinnerAddresses: []string{"addr1", "addr2"},
		VRFProof:        "0xproof-" + id,
		VRFOutput:       "0xoutput-" + id,
		Timestamp:       time.Now().Unix(),
		Verified:        true,
	}
}

// =================================================================
// JSON helpers
// =================================================================

func TestArrayToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"nil", nil, "null"},
		{"empty", []string{}, "[]"},
		{"single", []string{"a"}, `["a"]`},
		{"multi", []string{"a", "b", "c"}, `["a","b","c"]`},
		{"with-special-chars", []string{"a\"b", "c\nd"}, `["a\"b","c\nd"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, arrayToJSON(tt.in))
		})
	}
}

func TestJSONToArray(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"null", "null", nil},
		{"empty-array", "[]", []string{}},
		{"single", `["a"]`, []string{"a"}},
		{"multi", `["a","b","c"]`, []string{"a", "b", "c"}},
		{"invalid", "{not-json", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, jsonToArray(tt.in))
		})
	}
}

func TestArrayRoundTrip(t *testing.T) {
	originals := [][]string{
		nil,
		{},
		{"a"},
		{"alice", "bob", "carol", "dave", "eve"},
		{"with-dash", "with space", "with\"quote"},
	}
	for _, in := range originals {
		t.Run("", func(t *testing.T) {
			got := jsonToArray(arrayToJSON(in))
			if len(in) == 0 {
				// jsonToArray on "[]" returns []string{} not nil; both are "empty"
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, in, got)
		})
	}
}

// =================================================================
// Repository lifecycle
// =================================================================

func TestNewLotteryRepository_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "x.db")

	repo, err := NewLotteryRepository(dbPath)
	require.NoError(t, err)
	require.NotNil(t, repo)
	t.Cleanup(func() { _ = repo.Close() })

	assert.Equal(t, dbPath, repo.dbPath)
}

func TestNewLotteryRepository_CreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested path that does not yet exist; the constructor should
	// create the parent directory itself.
	dbPath := filepath.Join(tmpDir, "nested", "deeper", "lottery.db")

	repo, err := NewLotteryRepository(dbPath)
	require.NoError(t, err)
	require.NotNil(t, repo)
	t.Cleanup(func() { _ = repo.Close() })

	// Parent dir should exist.
	info, statErr := os.Stat(filepath.Dir(dbPath))
	require.NoError(t, statErr)
	assert.True(t, info.IsDir())
}

// =================================================================
// Save / GetByID
// =================================================================

func TestLotteryRepository_Save_AndGetByID(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	original := sampleLotteryRecord("L-1", 42)

	err := repo.Save(original)
	require.NoError(t, err)

	got, err := repo.GetByID("L-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, original.ID, got.ID)
	assert.Equal(t, original.BlockHeight, got.BlockHeight)
	assert.Equal(t, original.Seed, got.Seed)
	assert.Equal(t, original.Participants, got.Participants)
	assert.Equal(t, original.Winners, got.Winners)
	assert.Equal(t, original.WinnerAddresses, got.WinnerAddresses)
	assert.Equal(t, original.VRFProof, got.VRFProof)
	assert.Equal(t, original.VRFOutput, got.VRFOutput)
	assert.Equal(t, original.Timestamp, got.Timestamp)
	assert.Equal(t, original.Verified, got.Verified)
}

func TestLotteryRepository_Save_OverwritesExistingRecord(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	// Save initial record.
	r1 := sampleLotteryRecord("L-1", 10)
	r1.Winners = []string{"alice"}
	require.NoError(t, repo.Save(r1))

	// Re-save same id with different content.
	r2 := sampleLotteryRecord("L-1", 20)
	r2.Winners = []string{"bob", "carol"}
	require.NoError(t, repo.Save(r2))

	got, err := repo.GetByID("L-1")
	require.NoError(t, err)
	assert.Equal(t, int64(20), got.BlockHeight, "Save should overwrite (INSERT OR REPLACE)")
	assert.Equal(t, []string{"bob", "carol"}, got.Winners)
}

func TestLotteryRepository_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	got, err := repo.GetByID("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "lottery record not found")
}

func TestLotteryRepository_Save_UnverifiedFlag(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	r := sampleLotteryRecord("L-unverified", 1)
	r.Verified = false
	require.NoError(t, repo.Save(r))

	got, err := repo.GetByID("L-unverified")
	require.NoError(t, err)
	assert.False(t, got.Verified, "Verified=false should round-trip as false")
}

// =================================================================
// GetAll
// =================================================================

func TestLotteryRepository_GetAll_Empty(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	got, err := repo.GetAll()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLotteryRepository_GetAll_OrdersByTimestampDesc(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	base := time.Now().Unix()

	// Save records out of order; GetAll should return them newest-first.
	r1 := sampleLotteryRecord("A", 1)
	r1.Timestamp = base
	r2 := sampleLotteryRecord("B", 2)
	r2.Timestamp = base + 100 // newest
	r3 := sampleLotteryRecord("C", 3)
	r3.Timestamp = base + 50

	require.NoError(t, repo.Save(r1))
	require.NoError(t, repo.Save(r2))
	require.NoError(t, repo.Save(r3))

	got, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "B", got[0].ID, "newest timestamp should be first")
	assert.Equal(t, "C", got[1].ID)
	assert.Equal(t, "A", got[2].ID, "oldest timestamp should be last")
}

// =================================================================
// GetByBlockHeight
// =================================================================

func TestLotteryRepository_GetByBlockHeight_NoMatches(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	require.NoError(t, repo.Save(sampleLotteryRecord("L-1", 5)))

	got, err := repo.GetByBlockHeight(999)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLotteryRepository_GetByBlockHeight_FiltersCorrectly(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	require.NoError(t, repo.Save(sampleLotteryRecord("A", 10)))
	require.NoError(t, repo.Save(sampleLotteryRecord("B", 20)))
	require.NoError(t, repo.Save(sampleLotteryRecord("C", 10))) // same height as A
	require.NoError(t, repo.Save(sampleLotteryRecord("D", 30)))

	got, err := repo.GetByBlockHeight(10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	ids := []string{got[0].ID, got[1].ID}
	assert.Contains(t, ids, "A")
	assert.Contains(t, ids, "C")
}

// =================================================================
// Close
// =================================================================

func TestLotteryRepository_Close(t *testing.T) {
	repo, _ := setupLotteryTestDB(t)
	// Don't call cleanup; we'll call Close ourselves.
	require.NoError(t, repo.Close())

	// Calling Close again on a closed db should error (driver-dependent),
	// but never panic. We just assert no panic here.
	assert.NotPanics(t, func() {
		_ = repo.Close()
	})
}
