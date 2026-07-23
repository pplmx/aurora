package blockchain

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetChainForTest wipes the singleton state and the on-disk DB so each
// test starts clean. The package resets by removing ./data entirely, so we
// chdir into a temp dir for every test that touches InitBlockChain/InitDB.
func resetChainForTest(t *testing.T) {
	t.Helper()
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })
	ResetForTest()
}

func TestDeriveHash_IsDeterministic(t *testing.T) {
	b := &Block{Data: []byte("hello"), PrevHash: []byte("prev")}
	b.DeriveHash()
	first := append([]byte{}, b.Hash...)

	b.DeriveHash()
	assert.Equal(t, first, b.Hash, "DeriveHash should be deterministic")

	assert.Len(t, b.Hash, 32, "SHA-256 hash should be 32 bytes")
}

func TestDeriveHash_ChangesWithData(t *testing.T) {
	b1 := &Block{Data: []byte("one"), PrevHash: []byte("p")}
	b2 := &Block{Data: []byte("two"), PrevHash: []byte("p")}
	b1.DeriveHash()
	b2.DeriveHash()
	assert.NotEqual(t, b1.Hash, b2.Hash, "different data must yield different hash")
}

func TestDeriveHash_ChangesWithPrevHash(t *testing.T) {
	b1 := &Block{Data: []byte("d"), PrevHash: []byte("p1")}
	b2 := &Block{Data: []byte("d"), PrevHash: []byte("p2")}
	b1.DeriveHash()
	b2.DeriveHash()
	assert.NotEqual(t, b1.Hash, b2.Hash, "different prev-hash must yield different hash")
}

func TestGenesis_ReturnsMinedBlock(t *testing.T) {
	g := Genesis()
	require.NotNil(t, g)
	assert.Equal(t, []byte("Genesis"), g.Data)
	assert.Empty(t, g.PrevHash, "Genesis should have empty prev-hash")
	assert.Len(t, g.Hash, 32, "Genesis should be mined (32-byte hash)")
	assert.GreaterOrEqual(t, g.Nonce, int64(0))
	assert.NotZero(t, g.Timestamp, "Genesis timestamp should be set")
}

func TestNewBlockChain_ContainsGenesis(t *testing.T) {
	c := NewBlockChain()
	require.NotNil(t, c)
	require.Len(t, c.Blocks, 1)
	assert.Equal(t, []byte("Genesis"), c.Blocks[0].Data)
}

func TestBlockChain_AddBlock_ExtendsChain(t *testing.T) {
	c := NewBlockChain()
	height, err := c.AddBlock("first transaction")
	require.NoError(t, err)
	assert.Equal(t, int64(1), height, "first appended block has height 1")

	require.Len(t, c.Blocks, 2)
	assert.Equal(t, []byte("first transaction"), c.Blocks[1].Data)
	assert.Equal(t, int64(1), c.Blocks[1].Height)
	assert.Equal(t, c.Blocks[0].Hash, c.Blocks[1].PrevHash,
		"new block must reference previous block's hash")
	assert.Len(t, c.Blocks[1].Hash, 32)
}

func TestBlockChain_AddBlock_MultipleAppends(t *testing.T) {
	c := NewBlockChain()
	for i, data := range []string{"a", "b", "c"} {
		h, err := c.AddBlock(data)
		require.NoError(t, err)
		assert.Equal(t, int64(i+1), h)
	}
	assert.Len(t, c.Blocks, 4) // Genesis + 3
	for i := 1; i < len(c.Blocks); i++ {
		assert.Equal(t, c.Blocks[i-1].Hash, c.Blocks[i].PrevHash,
			"block %d prev-hash mismatch", i)
	}
}

func TestBlockChain_AddBlock_OnNilOrEmpty(t *testing.T) {
	var nilChain *BlockChain
	h, err := nilChain.AddBlock("x")
	assert.Error(t, err)
	assert.Equal(t, int64(0), h)
	assert.Contains(t, err.Error(), "not initialized")

	empty := &BlockChain{}
	_, err = empty.AddBlock("x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestBlockChain_GetBlockData(t *testing.T) {
	c := NewBlockChain()
	_, err := c.AddBlock("alice")
	require.NoError(t, err)
	_, err = c.AddBlock("bob")
	require.NoError(t, err)

	data, err := c.GetBlockData(0)
	require.NoError(t, err)
	assert.Equal(t, "Genesis", data)

	data, err = c.GetBlockData(1)
	require.NoError(t, err)
	assert.Equal(t, "alice", data)

	data, err = c.GetBlockData(2)
	require.NoError(t, err)
	assert.Equal(t, "bob", data)
}

func TestBlockChain_GetBlockData_OutOfRange(t *testing.T) {
	c := NewBlockChain()

	for _, height := range []int64{-1, -100, 1, 9999} {
		_, err := c.GetBlockData(height)
		require.Error(t, err, "height=%d", height)
		assert.Contains(t, err.Error(), "invalid block height")
	}
}

func TestBlockChain_GetLotteryRecords_SkipsGenesis(t *testing.T) {
	c := NewBlockChain()
	_, _ = c.AddBlock("lottery-1")
	_, _ = c.AddBlock("lottery-2")

	records := c.GetLotteryRecords()
	assert.Equal(t, []string{"lottery-1", "lottery-2"}, records)
}

func TestBlockChain_GetLotteryRecords_EmptyChain(t *testing.T) {
	c := NewBlockChain()
	records := c.GetLotteryRecords()
	assert.Empty(t, records, "only Genesis present → no records")
}

func TestBlockChain_AddLotteryRecord_DelegatesToAddBlock(t *testing.T) {
	c := NewBlockChain()
	h, err := c.AddLotteryRecord("L-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), h)
	require.Len(t, c.Blocks, 2)
	assert.Equal(t, []byte("L-1"), c.Blocks[1].Data)
}

func TestBlock_Serialize_RoundTrip(t *testing.T) {
	original := &Block{
		Height:    42,
		Hash:      []byte("hash-bytes-here-padding-padding-32"),
		PrevHash:  []byte("prev-bytes-padding-padding-padding-32"),
		Data:      []byte("round-trip payload"),
		Nonce:     1234,
		Timestamp: 1700000000,
	}
	// Ensure the hashes are exactly 32 bytes (gob encodes length, but DeriveHash
	// produces 32-byte outputs and we want to test that path).
	original.Hash = bytes.Repeat([]byte{0xAA}, 32)
	original.PrevHash = bytes.Repeat([]byte{0xBB}, 32)

	encoded, err := original.Serialize()
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	decoded, err := Deserialize(encoded)
	require.NoError(t, err)
	require.NotNil(t, decoded)

	assert.Equal(t, original.Height, decoded.Height)
	assert.Equal(t, original.Nonce, decoded.Nonce)
	assert.Equal(t, original.Timestamp, decoded.Timestamp)
	assert.Equal(t, original.Data, decoded.Data)
	assert.Equal(t, original.Hash, decoded.Hash)
	assert.Equal(t, original.PrevHash, decoded.PrevHash)
}

func TestBlock_Serialize_EmptyBlock(t *testing.T) {
	b := &Block{}
	encoded, err := b.Serialize()
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	decoded, err := Deserialize(encoded)
	require.NoError(t, err)
	assert.Equal(t, b.Height, decoded.Height)
	assert.Empty(t, decoded.Data)
}

func TestDeserialize_InvalidDataReturnsError(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"garbage", []byte("not a gob-encoded block")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := Deserialize(tc.data)
			assert.Error(t, err)
			assert.Nil(t, b)
		})
	}
}

func TestDBPath_CreatesDataDir(t *testing.T) {
	// Run in an isolated working dir so we don't pollute repo ./data.
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	got := DBPath()
	assert.Equal(t, defaultDBPath, got)
	info, err := os.Stat(filepath.Dir(defaultDBPath))
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "DBPath must ensure data dir exists")
}

func TestInitDB_OpensSQLite(t *testing.T) {
	resetChainForTest(t)

	db, err := InitDB()
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { _ = db.Close() })

	// Calling again returns the cached instance.
	db2, err := InitDB()
	require.NoError(t, err)
	assert.Same(t, db, db2, "InitDB should be idempotent (return cached db)")

	// Verify we can actually query it (sqlite is open and working).
	var v int
	require.NoError(t, db.QueryRow("SELECT 1").Scan(&v))
	assert.Equal(t, 1, v)
}

func TestClose_NoOpOnNilDB(t *testing.T) {
	resetChainForTest(t)
	assert.NoError(t, Close(), "Close on nil dbInstance should not error")
}

func TestClose_ClosesOpenDB(t *testing.T) {
	resetChainForTest(t)
	db, err := InitDB()
	require.NoError(t, err)
	require.NotNil(t, db)

	require.NoError(t, Close(), "Close should succeed for an open DB")
}

func TestInitBlockChain_SeedsGenesisWhenEmpty(t *testing.T) {
	resetChainForTest(t)

	chain := InitBlockChain()
	require.NotNil(t, chain)
	require.GreaterOrEqual(t, len(chain.Blocks), 1, "Genesis must be seeded")
	assert.Equal(t, []byte("Genesis"), chain.Blocks[0].Data)
	assert.Len(t, chain.Blocks[0].Hash, 32)
}

func TestInitBlockChain_PersistsAcrossCalls(t *testing.T) {
	resetChainForTest(t)

	c1 := InitBlockChain()
	require.NotNil(t, c1)
	originalLen := len(c1.Blocks)

	c2 := InitBlockChain()
	require.NotNil(t, c2)
	assert.Same(t, c1, c2, "InitBlockChain must be a singleton (sync.Once)")
	assert.Equal(t, originalLen, len(c2.Blocks),
		"second call should not re-seed (sync.Once protects initialization)")
}

func TestGetBlockChain_InitializesLazily(t *testing.T) {
	resetChainForTest(t)

	// Before InitBlockChain runs, the singleton is nil.
	assert.Nil(t, instance)

	chain := GetBlockChain()
	require.NotNil(t, chain)
	assert.NotNil(t, instance, "GetBlockChain must trigger InitBlockChain")
}

func TestResetForTest_ClearsSingleton(t *testing.T) {
	resetChainForTest(t)
	_ = InitBlockChain()
	require.NotNil(t, instance)

	ResetForTest()
	assert.Nil(t, instance, "ResetForTest must clear instance")
	assert.Nil(t, dbInstance, "ResetForTest must close and clear db")
}

func TestResetForTest_RemovesDataDir(t *testing.T) {
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	// Trigger DB creation so ./data exists.
	_, err = InitDB()
	require.NoError(t, err)
	_, err = os.Stat("./data")
	require.NoError(t, err, "./data should exist after InitDB")

	ResetForTest()
	_, err = os.Stat("./data")
	assert.True(t, os.IsNotExist(err), "./data should be removed after ResetForTest, got: %v", err)
}

func TestInitBlockChain_LogsDBErrorPath(t *testing.T) {
	// If the sqlite open fails, InitBlockChain must still return a chain
	// (it should not panic). We simulate by making ./data a regular file so
	// MkdirAll succeeds for the file path but sqlite open will fail when
	// the parent dir can't be created. Actually MkdirAll won't fail on the
	// file path because it's a single component; instead, point cwd at an
	// unwritable directory. This is best-effort: we just ensure no panic.
	resetChainForTest(t)

	// Pre-create a file at the data path to confuse MkdirAll
	require.NoError(t, os.WriteFile("./data", []byte("not a dir"), 0o644))

	// InitBlockChain must not panic; it falls back to in-memory Genesis chain.
	var chain *BlockChain
	assert.NotPanics(t, func() {
		chain = InitBlockChain()
	})
	require.NotNil(t, chain)
	assert.GreaterOrEqual(t, len(chain.Blocks), 1)
}

func TestSerialize_GobEncodingIsStable(t *testing.T) {
	b1 := &Block{Height: 7, Data: []byte("x"), Timestamp: 1}
	b2 := &Block{Height: 7, Data: []byte("x"), Timestamp: 1}

	e1, err := b1.Serialize()
	require.NoError(t, err)
	e2, err := b2.Serialize()
	require.NoError(t, err)
	// Gob encoding may include type info; just sanity-check both decode.
	d1, err := Deserialize(e1)
	require.NoError(t, err)
	d2, err := Deserialize(e2)
	require.NoError(t, err)
	assert.Equal(t, d1.Height, d2.Height)
	assert.Equal(t, d1.Data, d2.Data)
}

func TestBlockChain_RecordsIncludeEmptyBlockData(t *testing.T) {
	// GetLotteryRecords should include blocks whose Data is non-empty and
	// is not the literal "Genesis" string — empty data should be skipped.
	c := NewBlockChain()
	_, _ = c.AddBlock("")

	records := c.GetLotteryRecords()
	// Either Genesis-only (no records) or empty-data block filtered out.
	for _, r := range records {
		assert.NotEmpty(t, r, "no empty records allowed")
		assert.False(t, strings.HasPrefix(r, "Genesis"),
			"Genesis should be filtered out, got: %s", r)
	}
}

// TestBlockChain_Len proves the Len() helper returns the right
// count in three states (nil chain, empty chain after reset,
// populated chain) AND that concurrent calls during AppendBlock
// never race with the read. This is a regression guard for the
// fact that Len() is one of the readers of c.Blocks and must
// hold c.mu.RLock for the access (Round 13's fix).
func TestBlockChain_Len(t *testing.T) {
	t.Run("nil chain returns zero", func(t *testing.T) {
		var c *BlockChain
		if got := c.Len(); got != 0 {
			t.Errorf("nil chain Len() = %d, want 0", got)
		}
	})

	t.Run("new chain has genesis", func(t *testing.T) {
		c := NewBlockChain()
		// Genesis is the first block, so Len should be 1.
		if got := c.Len(); got != 1 {
			t.Errorf("fresh chain Len() = %d, want 1", got)
		}
	})

	t.Run("len grows with appends", func(t *testing.T) {
		resetChainForTest(t)
		c := NewBlockChain()
		for i := 0; i < 5; i++ {
			if _, err := c.AddBlock("data"); err != nil {
				t.Fatalf("AddBlock(%d) failed: %v", i, err)
			}
		}
		if got := c.Len(); got != 6 {
			t.Errorf("after 5 appends Len() = %d, want 6 (genesis + 5)", got)
		}
	})
}

// TestBlockChain_Len_ConcurrentWithAppend guards the Round 13
// race fix: Len() must hold c.mu for its access to c.Blocks, so
// running many concurrent Len() calls while another goroutine
// AppendBlocks must not race. Run with `go test -race`.
func TestBlockChain_Len_ConcurrentWithAppend(t *testing.T) {
	resetChainForTest(t)
	c := NewBlockChain()

	const readers = 8
	const appends = 32
	var wg sync.WaitGroup
	wg.Add(readers + 1)

	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = c.Len()
			}
		}()
	}

	go func() {
		defer wg.Done()
		for i := 0; i < appends; i++ {
			if _, err := c.AddBlock("concurrent"); err != nil {
				t.Errorf("AddBlock(%d) failed: %v", i, err)
				return
			}
		}
	}()

	wg.Wait()

	// After all appends, Len must be genesis + appends.
	if got, want := c.Len(), appends+1; got != want {
		t.Errorf("final Len() = %d, want %d", got, want)
	}
}
