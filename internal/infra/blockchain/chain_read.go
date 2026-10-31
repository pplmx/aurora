// Reading the chain: the block-entry views the ledger page and the search path
// consume, paged or filtered, plus the length the pager bounds itself by. They
// are projections of the in-memory blocks, so they share the chain's read lock
// and never touch the database.
package blockchain

import (
	"github.com/pplmx/aurora/internal/ledger"
)

// BlockEntry is the pure ledger type; the alias keeps the chain's public API
// stable for callers that still name blockchain.BlockEntry.
type BlockEntry = ledger.BlockEntry

// BlockEntries returns the non-genesis payloads newest first, at most limit of
// them (limit <= 0 returns every payload), each paired with its block's own
// height. Callers that only show the latest few avoid materialising the whole
// chain, and skipping a payload can never shift a height: the number comes
// from the block, not from the payload's position in a filtered slice.
func (c *Chain) BlockEntries(limit int) []BlockEntry {
	return c.BlockEntriesPage(limit, 0)
}

// BlockEntriesPage returns a page of non-genesis payloads newest first. offset
// is in entries, not block heights: the page can skip a long ledger without
// materialising every payload before the one it shows. limit <= 0 returns all
// entries from offset.
func (c *Chain) BlockEntriesPage(limit, offset int) []BlockEntry {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if offset < 0 {
		offset = 0
	}
	capacity := len(c.Blocks)
	if limit > 0 {
		capacity = min(limit, capacity)
	}
	entries := make([]BlockEntry, 0, capacity)
	seen := 0
	for i := len(c.Blocks) - 1; i >= 0; i-- {
		block := c.Blocks[i]
		data := string(block.Data)
		if data == "" || data == "Genesis" {
			continue
		}
		if seen < offset {
			seen++
			continue
		}
		entries = append(entries, BlockEntry{Height: block.Height, Timestamp: block.Timestamp, Data: data})
		if limit > 0 && len(entries) == limit {
			break
		}
	}
	return entries
}

// BlockEntriesPageMatching returns the matching entries in one newest-first
// page and the total number of matches. It still scans the whole chain to make
// the total exact, but only the requested page is retained; callers searching
// a long ledger no longer need to materialise every row before slicing it.
// A nil match keeps every non-genesis entry.
func (c *Chain) BlockEntriesPageMatching(limit, offset int, match func(BlockEntry) bool) ([]BlockEntry, int) {
	if c == nil {
		return nil, 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if offset < 0 {
		offset = 0
	}
	capacity := len(c.Blocks)
	if limit > 0 {
		capacity = min(limit, capacity)
	}
	entries := make([]BlockEntry, 0, capacity)
	matched := 0
	for i := len(c.Blocks) - 1; i >= 0; i-- {
		block := c.Blocks[i]
		data := string(block.Data)
		if data == "" || data == "Genesis" {
			continue
		}
		entry := BlockEntry{Height: block.Height, Timestamp: block.Timestamp, Data: data}
		if match != nil && !match(entry) {
			continue
		}
		matched++
		if matched <= offset {
			continue
		}
		if limit > 0 && len(entries) >= limit {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, matched
}

func (c *Chain) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Blocks)
}
