package blockchain

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetChainForTest wipes the singleton state and the on-disk DB so each
// test starts clean. The package resets by removing ./data entirely, so we
// chdir into a temp dir for every test that touches Init/InitDB.
func resetChainForTest(t *testing.T) {
	t.Helper()
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prevDir) })
	// Close the process-wide DB singleton before t.TempDir()'s RemoveAll so
	// the SQLite handle isn't left open (Windows can't delete an in-use file;
	// Unix tolerates unlink-of-open-file, which is why this was invisible).
	t.Cleanup(func() { _ = Close() })
	ResetForTest()
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

func TestNewChain_ContainsGenesis(t *testing.T) {
	c := New()
	require.NotNil(t, c)
	require.Len(t, c.Blocks, 1)
	assert.Equal(t, []byte("Genesis"), c.Blocks[0].Data)
}

func TestChain_AddBlock_ExtendsChain(t *testing.T) {
	c := New()
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

func TestChain_AddBlock_MultipleAppends(t *testing.T) {
	c := New()
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

func TestChain_AddBlock_OnNilOrEmpty(t *testing.T) {
	var nilChain *Chain
	h, err := nilChain.AddBlock("x")
	assert.Error(t, err)
	assert.Equal(t, int64(0), h)
	assert.Contains(t, err.Error(), "not initialized")

	empty := &Chain{}
	_, err = empty.AddBlock("x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestChain_BlockData_SkipsGenesis(t *testing.T) {
	c := New()
	_, _ = c.AddBlock("lottery-1")
	_, _ = c.AddBlock("lottery-2")

	records := c.BlockData()
	assert.Equal(t, []string{"lottery-1", "lottery-2"}, records)
}

func TestChain_BlockData_EmptyChain(t *testing.T) {
	c := New()
	records := c.BlockData()
	assert.Empty(t, records, "only Genesis present → no records")
}

func TestChain_AddRecord_DelegatesToAddBlock(t *testing.T) {
	c := New()
	h, err := c.AddRecord("L-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), h)
	require.Len(t, c.Blocks, 2)
	assert.Equal(t, []byte("L-1"), c.Blocks[1].Data)
}

func TestChain_RecordsIncludeEmptyBlockData(t *testing.T) {
	// BlockData should include blocks whose Data is non-empty and
	// is not the literal "Genesis" string — empty data should be skipped.
	c := New()
	_, _ = c.AddBlock("")

	records := c.BlockData()
	// Either Genesis-only (no records) or empty-data block filtered out.
	for _, r := range records {
		assert.NotEmpty(t, r, "no empty records allowed")
		assert.False(t, strings.HasPrefix(r, "Genesis"),
			"Genesis should be filtered out, got: %s", r)
	}
}

// TestChain_Len proves the Len() helper returns the right
// count in three states (nil chain, empty chain after reset,
// populated chain) AND that concurrent calls during AppendBlock
// never race with the read. This is a regression guard for the
// fact that Len() is one of the readers of c.Blocks and must
// hold c.mu.RLock for the access (Round 13's fix).
func TestChain_Len(t *testing.T) {
	t.Run("nil chain returns zero", func(t *testing.T) {
		var c *Chain
		if got := c.Len(); got != 0 {
			t.Errorf("nil chain Len() = %d, want 0", got)
		}
	})

	t.Run("new chain has genesis", func(t *testing.T) {
		c := New()
		// Genesis is the first block, so Len should be 1.
		if got := c.Len(); got != 1 {
			t.Errorf("fresh chain Len() = %d, want 1", got)
		}
	})

	t.Run("len grows with appends", func(t *testing.T) {
		resetChainForTest(t)
		c := New()
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

// TestChain_ResetBlocks_ReseedsToGenesis locks the v1.58 fix: after
// appends, ResetBlocks() must collapse the in-memory chain back to a single
// genesis block so the next AddBlock restarts at height 1 (and links to
// genesis), not the stale pre-reset height. This is what `lottery reset`
// relies on after clearing the persisted tables (TASK-071, ISS-063).
func TestChain_ResetBlocks_ReseedsToGenesis(t *testing.T) {
	resetChainForTest(t)
	c := New()
	for i := 0; i < 4; i++ {
		if _, err := c.AddBlock("data"); err != nil {
			t.Fatalf("AddBlock(%d) failed: %v", i, err)
		}
	}
	if got := c.Len(); got != 5 {
		t.Fatalf("before reset Len() = %d, want 5 (genesis + 4)", got)
	}

	c.ResetBlocks()

	if got := c.Len(); got != 1 {
		t.Fatalf("after reset Len() = %d, want 1 (genesis only)", got)
	}
	// The next append restarts at height 1 and links to genesis.
	h, err := c.AddBlock("fresh")
	if err != nil {
		t.Fatalf("AddBlock after reset failed: %v", err)
	}
	if h != 1 {
		t.Fatalf("post-reset AddBlock height = %d, want 1", h)
	}
	if got := c.Blocks[1].PrevHash; !bytes.Equal(got, c.Blocks[0].Hash) {
		t.Errorf("post-reset block PrevHash does not link to genesis")
	}
}

// TestChain_Len_ConcurrentWithAppend guards the Round 13
// race fix: Len() must hold c.mu for its access to c.Blocks, so
// running many concurrent Len() calls while another goroutine
// AppendBlocks must not race. Run with `go test -race`.
func TestChain_Len_ConcurrentWithAppend(t *testing.T) {
	resetChainForTest(t)
	c := New()

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

func TestVerifyIntegrity_ValidChain(t *testing.T) {
	c := New()
	_, err := c.AddBlock("first")
	require.NoError(t, err)
	_, err = c.AddBlock("second")
	require.NoError(t, err)

	rep := c.VerifyIntegrity()
	require.True(t, rep.Valid, "valid chain must verify: %+v", rep)
	require.Equal(t, 3, rep.Length) // genesis + 2
	require.Equal(t, -1, rep.FirstBrokenIndex)
}

func TestVerifyIntegrity_DetectsPrevHashBreak(t *testing.T) {
	c := New()
	_, err := c.AddBlock("a")
	require.NoError(t, err)
	_, err = c.AddBlock("b")
	require.NoError(t, err)

	// Corrupt the second added block's PrevHash (break linkage).
	c.mu.Lock()
	c.Blocks[1].PrevHash = []byte("tampered")
	c.mu.Unlock()

	rep := c.VerifyIntegrity()
	require.False(t, rep.Valid)
	require.Equal(t, 1, rep.FirstBrokenIndex)
}

func TestVerifyIntegrity_DetectsTamperedHash(t *testing.T) {
	c := New()
	_, err := c.AddBlock("a")
	require.NoError(t, err)

	c.mu.Lock()
	c.Blocks[1].Hash = []byte("forged")
	c.mu.Unlock()

	rep := c.VerifyIntegrity()
	require.False(t, rep.Valid)
	require.Equal(t, 1, rep.FirstBrokenIndex)
}

// TestBlockEntriesCarryTheirHeights pins the ledger's height source. Payloads
// are filtered (genesis has none), so deriving a height from a payload's index
// showed every block one too low — the desktop ledger read height 0 for the
// first recorded block.
func TestBlockEntriesCarryTheirHeights(t *testing.T) {
	c := New()
	_, err := c.AddBlock("first")
	require.NoError(t, err)
	_, err = c.AddBlock("second")
	require.NoError(t, err)

	entries := c.BlockEntries(0)
	require.Len(t, entries, 2)
	require.Equal(t, int64(2), entries[0].Height)
	require.Equal(t, "second", entries[0].Data)
	require.Equal(t, int64(1), entries[1].Height)
	require.Equal(t, "first", entries[1].Data)

	latest := c.BlockEntries(1)
	require.Len(t, latest, 1)
	require.Equal(t, int64(2), latest[0].Height)
}

// TestBlockEntriesPageOffsetsInEntries keeps pagination from confusing a block
// height with a position: offsets skip non-genesis entries, newest first.
func TestBlockEntriesPageOffsetsInEntries(t *testing.T) {
	chain := New()
	for _, payload := range []string{"a", "b", "c"} {
		_, err := chain.AddBlock(payload)
		require.NoError(t, err)
	}
	page := chain.BlockEntriesPage(1, 1)
	require.Len(t, page, 1)
	require.Equal(t, "b", page[0].Data)
	require.Equal(t, int64(2), page[0].Height)
	require.Empty(t, chain.BlockEntriesPage(1, 9))
}

// TestBlockEntriesPageMatchingPagesAndCountsAllMatches keeps a filtered page
// bounded while still returning the exact total needed by the pager. The scan
// must continue past the requested page so later matches affect that total.
func TestBlockEntriesPageMatchingPagesAndCountsAllMatches(t *testing.T) {
	chain := New()
	for _, payload := range []string{
		"needle-a", "other", "needle-b", "other", "needle-c", "needle-d",
	} {
		_, err := chain.AddBlock(payload)
		require.NoError(t, err)
	}

	matchesNeedle := func(entry BlockEntry) bool {
		return strings.HasPrefix(entry.Data, "needle-")
	}
	page, total := chain.BlockEntriesPageMatching(2, 1, matchesNeedle)
	require.Equal(t, 4, total, "the total must count matches beyond the page")
	require.Len(t, page, 2)
	require.Equal(t, "needle-c", page[0].Data, "matches stay newest first")
	require.Equal(t, int64(5), page[0].Height)
	require.Equal(t, "needle-b", page[1].Data)
	require.Equal(t, int64(3), page[1].Height)

	all, total := chain.BlockEntriesPageMatching(0, 0, nil)
	require.Equal(t, 6, total)
	require.Len(t, all, 6)
}
