package sqlite

import (
	"database/sql"
	"fmt"
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
		VRFPublicKey:    "public-key-" + id,
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
	assert.Equal(t, original.VRFPublicKey, got.VRFPublicKey)
	assert.Equal(t, original.Timestamp, got.Timestamp)
	assert.Equal(t, original.Verified, got.Verified)
}

func TestLotteryRepository_VRFPublicKeyOnEveryReadPath(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	original := sampleLotteryRecord("L-keyed", 42)
	require.NoError(t, repo.Save(original))

	byID, err := repo.GetByID(original.ID)
	require.NoError(t, err)
	assert.Equal(t, original.VRFPublicKey, byID.VRFPublicKey)

	all, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, original.VRFPublicKey, all[0].VRFPublicKey)

	byHeight, err := repo.GetByBlockHeight(original.BlockHeight)
	require.NoError(t, err)
	require.Len(t, byHeight, 1)
	assert.Equal(t, original.VRFPublicKey, byHeight[0].VRFPublicKey)
}

func TestLotteryRepository_UpgradesLegacySchemaWithVRFKeyTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", DSN(dbPath))
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE lottery_records (
			id TEXT PRIMARY KEY,
			block_height INTEGER NOT NULL,
			seed TEXT NOT NULL,
			participants TEXT NOT NULL,
			winners TEXT NOT NULL,
			winner_addresses TEXT NOT NULL,
			vrf_proof TEXT NOT NULL,
			vrf_output TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			verified INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO lottery_records
			(id, block_height, seed, participants, winners, winner_addresses,
			 vrf_proof, vrf_output, timestamp, verified)
		VALUES ('legacy', 1, 'legacy-seed', '["alice"]', '["alice"]', '["addr"]',
			'proof', 'output', 1, 0);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	repo, err := NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	legacy, err := repo.GetByID("legacy")
	require.NoError(t, err)
	assert.Empty(t, legacy.VRFPublicKey, "legacy draws have no recoverable key")

	keyed := sampleLotteryRecord("keyed-after-upgrade", 2)
	require.NoError(t, repo.Save(keyed), "the self-bootstrap must add the key table")
	got, err := repo.GetByID(keyed.ID)
	require.NoError(t, err)
	assert.Equal(t, keyed.VRFPublicKey, got.VRFPublicKey)
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

func TestLotteryRepository_ListPages(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	for i, timestamp := range []int64{100, 200, 300} {
		record := sampleLotteryRecord(fmt.Sprintf("page-%d", i), int64(i+1))
		record.Timestamp = timestamp
		require.NoError(t, repo.Save(record))
	}

	page, err := repo.List(1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "page-1", page[0].ID, "the second row in newest-first order")

	remainder, err := repo.List(0, 1)
	require.NoError(t, err)
	require.Len(t, remainder, 2, "limit zero keeps the full remainder from offset")

	past, err := repo.List(10, 99)
	require.NoError(t, err)
	require.Empty(t, past)
}

// TestLotteryRepository_GetAll_SameSecondIsNewestHeightFirst pins the
// tiebreaker. Draws made in the same second share a timestamp, and ordering by
// timestamp alone returned them in insertion order: the index walk is by
// rowid, so an older draw imported after a newer one came first and a page
// window could repeat or skip a row between requests.
func TestLotteryRepository_GetAll_SameSecondIsNewestHeightFirst(t *testing.T) {
	repo, cleanup := setupLotteryTestDB(t)
	defer cleanup()

	ts := time.Now().Unix()
	newer := sampleLotteryRecord("same-second-newer", 2)
	newer.Timestamp = ts
	older := sampleLotteryRecord("same-second-older", 1)
	older.Timestamp = ts

	// Save the newer draw first so insertion order (rowid) and chain order
	// disagree — exactly what an import of older draws into a live database
	// produces.
	require.NoError(t, repo.Save(newer))
	require.NoError(t, repo.Save(older))

	got, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, newer.ID, got[0].ID, "the later block height is the newer draw")
	assert.Equal(t, older.ID, got[1].ID)

	// A second read must return the same order, or offset paging over two
	// requests could show one draw twice and never show another.
	again, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, again, 2)
	assert.Equal(t, got[0].ID, again[0].ID)
	assert.Equal(t, got[1].ID, again[1].ID)
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
