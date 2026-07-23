package blockchain

import (
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
