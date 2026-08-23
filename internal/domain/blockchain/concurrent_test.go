package blockchain

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

// TestBlockChain_ConcurrentAddBlock_DataRace is a regression test that
// proves the BlockChain singleton is unsafe for concurrent AddBlock.
// Running with `go test -race` must surface the race.
func TestBlockChain_ConcurrentAddBlock_DataRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race-detection stress test in short mode")
	}
	ResetForTest()
	chain := InitBlockChain()

	const goroutines = 8
	const addsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < addsPerGoroutine; i++ {
				_, _ = chain.AddBlock(fmt.Sprintf("g%d-i%d", id, i))
			}
		}(g)
	}
	wg.Wait()

	// After concurrent additions, the chain length must be exactly
	// 1 (genesis) + goroutines*addsPerGoroutine. If AddBlock has a
	// race, slice append under contention will drop blocks.
	want := 1 + goroutines*addsPerGoroutine
	if got := len(chain.Blocks); got != want {
		t.Errorf("chain length after concurrent AddBlock = %d, want %d (raced append lost blocks)", got, want)
	}
}

// TestBlockChain_ConcurrentAddBlock_ChainIntegrity is a regression test for a
// semantic (not just data-race) bug: the old AddBlock read height/prevHash
// under a read lock, mined outside the lock, then appended under a write lock.
// That is a non-atomic read-modify-write, so concurrent callers reused the
// same tail and produced blocks with duplicate heights, gaps, and a broken
// prev-hash chain — and, because height is the DB primary key, persistence
// dropped a block per collision on restart. The length-only race test never
// caught this, so this test asserts chain validity directly.
func TestBlockChain_ConcurrentAddBlock_ChainIntegrity(t *testing.T) {
	chain := NewBlockChain()
	const goroutines = 8
	const addsPerGoroutine = 12

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < addsPerGoroutine; i++ {
				if _, err := chain.AddBlock(fmt.Sprintf("g%d-i%d", id, i)); err != nil {
					t.Errorf("AddBlock: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	wantLen := 1 + goroutines*addsPerGoroutine
	if got := len(chain.Blocks); got != wantLen {
		t.Fatalf("chain length = %d, want %d", got, wantLen)
	}

	seen := make(map[int64]bool, wantLen)
	for i, b := range chain.Blocks {
		if b.Height != int64(i) {
			t.Errorf("block[%d].Height = %d, want %d (duplicate/gap)", i, b.Height, i)
		}
		if seen[b.Height] {
			t.Errorf("duplicate height %d at slot %d", b.Height, i)
		}
		seen[b.Height] = true
		if i > 0 && !bytes.Equal(b.PrevHash, chain.Blocks[i-1].Hash) {
			t.Errorf("block[%d].PrevHash does not match previous block hash", i)
		}
		if len(b.Hash) == 0 {
			t.Errorf("block[%d] has empty hash (mined block not filled in)", i)
		}
	}
}
