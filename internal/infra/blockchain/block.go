// Package blockchain provides core blockchain infrastructure including
// block, blockchain, and proof-of-work implementations.
package blockchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pplmx/aurora/internal/ledger"
	"github.com/pplmx/aurora/internal/logger"
)

// Block represents a single block in the blockchain.
type Block struct {
	Height    int64
	Hash      []byte
	PrevHash  []byte
	Data      []byte
	Nonce     int64
	Timestamp int64
}

// Chain represents the blockchain data structure.
//
// Concurrency: Blocks is mutated only under mu. Reads (GetBlockData,
// BlockData) also take mu (RLock) so HTTP handlers can read
// safely while other goroutines are appending blocks via AddBlock.
// Without this mutex, a real production server with concurrent API
// traffic lost blocks and tripped the race detector (see
// TestChain_ConcurrentAddBlock_DataRace for the regression).
type Chain struct {
	mu     sync.RWMutex
	Blocks []*Block
	// persist, when non-nil, is invoked after a block is appended so the
	// chain can survive process restarts. It is left nil by New
	// (pure in-memory use, e.g. tests) and wired to the blocks table by
	// Init. The in-memory slice stays authoritative; a persistence
	// failure is logged rather than fatal, matching the package's existing
	// "operate in non-persistent mode" fallback.
	//
	// The persist hook must not silently overwrite a committed block: when the
	// backing store already holds a conflicting block at the same height (a
	// second process sharing the DB file reserved it first), persist must
	// return ErrHeightConflict so appendBlock can drop the losing candidate and
	// retry at the DB-authoritative next height (TASK-244, ISS-242).
	persist func(*Block) error

	// removePersist compensates for a best-effort persist in the nil-tx
	// fallback when the business callback or transaction manager then fails.
	// It must delete only the exact block it is given (height and hash), never
	// a different block a peer may have reserved at the same height.
	removePersist func(*Block) error

	// syncFromDB, when non-nil, is invoked under the write lock *before* the
	// next height is reserved. It brings the in-memory chain up to date with
	// blocks a concurrent process committed to the shared DB file, so the
	// reserved height is DB-authoritative (never a stale len(c.Blocks)).
	// Left nil by New; wired by Init.
	syncFromDB func() error
}

// ErrHeightConflict is returned by the persist hook when the backing store
// already contains a different block at the candidate height (a concurrent
// process won the race for that height). appendBlock drops the losing in-memory
// candidate and retries at the next free height instead of overwriting the
// committed block (TASK-244, ISS-242).
var ErrHeightConflict = errors.New("blockchain: block height already committed by another process")

// maxHeightRetries bounds the cross-process height-conflict retry loop in
// appendBlock. Each retry re-syncs from the DB (so it picks the true next
// height) and re-mines; the bound only trips under pathological multi-writer
// contention where even repeated fresh heights keep tying.
const maxHeightRetries = 5

func NewBlock(data string, prevHash []byte) *Block {
	block := &Block{
		Height:    0,
		Hash:      []byte{},
		PrevHash:  prevHash,
		Data:      []byte(data),
		Nonce:     0,
		Timestamp: time.Now().Unix(),
	}
	pow := NewProofOfWork(block)
	nonce, hash, _ := pow.Run(context.Background())

	block.Hash = hash[:]
	block.Nonce = int64(nonce)

	return block
}

func Genesis() *Block {
	return NewBlock("Genesis", []byte{})
}

// AddBlock appends a new block to the chain. Safe for concurrent callers.
//
// The reserve + mine + append sequence runs entirely under the write lock.
// This is intentional: mining is *not* parallelizable across blocks. A block's
// PrevHash must equal the finalized hash of the preceding block, and PrevHash
// feeds into the proof-of-work input, so block i's PoW cannot even begin until
// block i-1 is mined and its hash is known. Attempting to mine in parallel
// (reserving a slot, mining outside the lock, then filling in the hash) breaks
// chain integrity: a caller reserving slot i reads the previous slot while it
// is still an unmined placeholder and produces an invalid PrevHash.
//
// The historical implementation did exactly that (read height/prevHash under a
// read lock, mine outside, append under a write lock), which was a non-atomic
// read-modify-write: concurrent callers reused the same tail, yielding blocks
// with duplicate heights, gaps, and a broken prev-hash chain. Because height
// is the DB primary key, persistence also used INSERT OR REPLACE and silently
// dropped a block per collision on restart. Serializing the whole append under
// the write lock makes heights unique and the chain valid regardless of how
// many goroutines call AddBlock concurrently.
//
// Cross-process note (TASK-244, ISS-242): the in-process lock only serializes
// the callers of *this* process. Two processes sharing one DB file each have a
// process-local chain, so even the serialized reserve can pick a stale height.
// appendBlock therefore re-syncs the in-memory chain from the shared DB before
// reserving (syncFromDB) and detects/recovers from a lost height race via the
// persist hook returning ErrHeightConflict. A committed block is never
// overwritten; the losing payload is re-appended at the DB-authoritative next
// height.
func (c *Chain) AddBlock(data string) (int64, error) {
	return c.appendBlock(data, nil)
}

// appendBlock serializes the whole reserve+mine+append under the write lock
// (see AddBlock's comment for why mining cannot be parallelized). When stamp
// is non-nil it is called with the block height just before the block is
// mined, so callers can finalize the payload (e.g. stamp a self-describing
// height into an audit record's JSON) before Data becomes part of the PoW
// hash. A stamp that returns "" leaves data unchanged.
func (c *Chain) appendBlock(data string, stamp func(height int64) string) (int64, error) {
	if c == nil {
		return 0, fmt.Errorf("blockchain not initialized")
	}

	// The reserve + mine + append + persist sequence runs entirely under the
	// write lock. Persisting under the lock (rather than after unlock) is what
	// makes the cross-process retry loop safe: a concurrent AddBlock caller in
	// this process cannot interleave between an append and its persist, so a
	// detected ErrHeightConflict can always be resolved by dropping the block we
	// just appended (TASK-244, ISS-242). The mining cost dominates anyway; a
	// short single-row write adds negligible lock hold time.
	c.mu.Lock()
	defer c.mu.Unlock()

	for attempt := 0; attempt < maxHeightRetries; attempt++ {
		// Re-sync from the shared DB before reserving, so the height we pick is
		// the DB-authoritative next height even when a concurrent process has
		// appended blocks this process has not seen yet. Failure degrades to
		// the stale in-memory height (best-effort, matching the non-persistent
		// fallback); the persist conflict check below is the safety net.
		if err := c.syncBeforeAppend(); err != nil {
			return 0, err
		}

		block, err := c.nextBlock(data, stamp)
		if err != nil {
			return 0, err
		}
		height := block.Height
		c.Blocks = append(c.Blocks, block)

		// Best-effort persistence. The in-memory chain is authoritative;
		// non-conflict failures degrade to non-persistent mode.
		if c.persist == nil {
			return height, nil
		}
		if err := c.persist(block); err != nil {
			if errors.Is(err, ErrHeightConflict) {
				// A concurrent process committed this exact height first (or we
				// were re-sync'd past it). Our candidate must NOT overwrite it:
				// drop the block we appended this attempt and retry at the next
				// free height. Bounded so a pathological multi-writer battle
				// cannot loop forever; a fresh syncFromDB on the next attempt
				// yields the true next height.
				c.Blocks = c.Blocks[:len(c.Blocks)-1]
				logger.Warn().Int64("height", height).Msg("Block height already committed by another process; retrying at next height")
				continue
			}
			logger.Warn().Err(err).Msg("Failed to persist block")
		}

		return height, nil
	}

	return 0, ErrHeightConflict
}

// WithBlockTransaction appends data while running fn in the same SQLite
// transaction. The transaction inserts the block before fn runs, so a failed
// business mutation rolls back both rows and the in-memory chain never sees a
// phantom block.
//
// The chain write lock is held from height reservation through commit. This is
// required for two reasons: consecutive proof-of-work inputs cannot be mined
// in parallel, and no other caller may observe or extend the chain between a
// successful commit and the in-memory append. The lock order is always
// chain -> database, matching AddBlock, so transaction callbacks must not call
// back into this chain.
func (c *Chain) WithBlockTransaction(
	data string,
	txManager ledger.TransactionManager,
	fn func(tx *sql.Tx, height int64) error,
) error {
	return c.withAppendTransaction(data, nil, txManager, fn)
}

// WithRecordTransaction is WithBlockTransaction with the same
// block_height stamping used by AddRecord.
func (c *Chain) WithRecordTransaction(
	data string,
	txManager ledger.TransactionManager,
	fn func(tx *sql.Tx, height int64) error,
) error {
	return c.withAppendTransaction(data, func(height int64) string {
		stamped, err := stampRecordHeight(data, height)
		if err != nil {
			return ""
		}
		return stamped
	}, txManager, fn)
}

func (c *Chain) withAppendTransaction(
	data string,
	stamp func(height int64) string,
	txManager ledger.TransactionManager,
	fn func(tx *sql.Tx, height int64) error,
) error {
	if c == nil {
		return fmt.Errorf("blockchain not initialized")
	}
	if txManager == nil {
		return fmt.Errorf("blockchain transaction manager not configured")
	}
	if fn == nil {
		return fmt.Errorf("blockchain transaction callback not configured")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for range maxHeightRetries {
		if err := c.syncBeforeAppend(); err != nil {
			return err
		}

		block, err := c.nextBlock(data, stamp)
		if err != nil {
			return err
		}

		var persistedNoOp bool
		err = txManager.WithTransaction(func(tx *sql.Tx) error {
			if tx == nil {
				// A no-op/mock transaction cannot insert the block atomically.
				// Preserve the legacy best-effort behavior for in-memory
				// repositories, but still append the in-memory block only
				// after the callback succeeds.
				if c.persist != nil {
					if persistErr := c.persist(block); persistErr != nil {
						if errors.Is(persistErr, ErrHeightConflict) {
							return persistErr
						}
						logger.Warn().Err(persistErr).Msg("Failed to persist block")
					} else {
						persistedNoOp = true
					}
				}
				return fn(nil, block.Height)
			}

			if insertErr := insertBlock(tx, block); insertErr != nil {
				return insertErr
			}
			return fn(tx, block.Height)
		})
		if err != nil {
			if persistedNoOp {
				if c.removePersist == nil {
					rollbackErr := errors.New("blockchain: persistent block rollback not configured")
					logger.Error().Err(rollbackErr).Int64("height", block.Height).Msg("Failed to roll back persisted no-op transaction block")
					return errors.Join(err, rollbackErr)
				}
				if rollbackErr := c.removePersist(block); rollbackErr != nil {
					logger.Error().Err(rollbackErr).Int64("height", block.Height).Msg("Failed to roll back persisted no-op transaction block")
					return errors.Join(err, fmt.Errorf("blockchain: roll back persisted block at height %d: %w", block.Height, rollbackErr))
				}
			}
			if errors.Is(err, ErrHeightConflict) {
				logger.Warn().Int64("height", block.Height).Msg("Block height already committed by another process; retrying at next height")
				continue
			}
			return err
		}

		c.Blocks = append(c.Blocks, block)
		return nil
	}

	return ErrHeightConflict
}

// syncBeforeAppend reconciles with the shared database before reserving a
// height. Ordinary read failures retain the historical best-effort fallback,
// but persisted-ledger corruption is fatal: appending on top of it would
// report success while leaving a ledger that cannot pass the next reload.
// The caller must hold the chain write lock.
func (c *Chain) syncBeforeAppend() error {
	if c.syncFromDB == nil {
		return nil
	}
	if err := c.syncFromDB(); err != nil {
		if errors.Is(err, ErrLedgerCorruption) {
			return fmt.Errorf("refuse append with corrupt persisted ledger: %w", err)
		}
		logger.Warn().Err(err).Msg("Failed to sync chain from database; using in-memory height")
	}
	return nil
}

// nextBlock reserves and mines the next chain block. The caller must hold the
// chain write lock.
func (c *Chain) nextBlock(data string, stamp func(height int64) string) (*Block, error) {
	if len(c.Blocks) == 0 {
		return nil, fmt.Errorf("blockchain not initialized")
	}

	height := int64(len(c.Blocks))
	if stamp != nil {
		if stamped := stamp(height); stamped != "" {
			data = stamped
		}
	}
	prev := c.Blocks[len(c.Blocks)-1]
	block := &Block{
		Height:    height,
		PrevHash:  append([]byte(nil), prev.Hash...),
		Data:      []byte(data),
		Nonce:     0,
		Timestamp: time.Now().Unix(),
	}
	pow := NewProofOfWork(block)
	nonce, hash, _ := pow.Run(context.Background())
	block.Nonce = int64(nonce)
	block.Hash = append([]byte(nil), hash...)
	return block, nil
}

// ResetBlocks re-seeds the in-memory chain back to a single genesis block.
// It is used by `lottery reset` after the persisted `blocks` and
// `lottery_records` tables have been cleared: the chain singleton is created
// once (init.go's once.Do) and otherwise keeps the pre-reset blocks in memory,
// so a post-reset AddBlock would compute a stale height and a PrevHash that
// references a block the reset deleted. Re-seeding makes the next AddBlock
// start at height 1 with a valid genesis PrevHash (TASK-071, ISS-063).
func (c *Chain) ResetBlocks() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Blocks = []*Block{Genesis()}
}

// BlockData returns the payload of every non-genesis block, in height order.
// It is the read side of the audit trail: callers that want to inspect what was
// recorded without knowing each module's row schema use this.
func (c *Chain) BlockData() []string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	records := make([]string, 0, len(c.Blocks))
	for _, block := range c.Blocks {
		data := string(block.Data)
		if len(data) > 0 && data != "Genesis" {
			records = append(records, data)
		}
	}
	return records
}

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
		entries = append(entries, BlockEntry{Height: block.Height, Data: data})
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
		entry := BlockEntry{Height: block.Height, Data: data}
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

// IntegrityReport is the pure ledger type; the alias keeps VerifyIntegrity's
// public return type stable.
type IntegrityReport = ledger.IntegrityReport

// VerifyIntegrity cryptographically verifies the whole chain: heights must be
// contiguous from genesis, every block must carry a non-empty hash, each
// block's PrevHash must link to the previous block's hash, and each block's
// stored hash must be a valid proof-of-work over its fields. It returns a
// report with the first broken index (-1 if valid).
//
// This is the tamper-evidence surface behind the "blockchain-based" guarantees
// of the module (v1.25): operators can prove the ledger is internally consistent.
func (c *Chain) VerifyIntegrity() *IntegrityReport {
	if c == nil {
		return &IntegrityReport{Valid: false, FirstBrokenIndex: 0, FirstBrokenReason: "nil chain"}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	blocks := c.Blocks
	n := len(blocks)
	for i, b := range blocks {
		if b == nil {
			return &IntegrityReport{Valid: false, Length: n, FirstBrokenIndex: i, FirstBrokenReason: "nil block"}
		}
		if b.Height != int64(i) {
			return &IntegrityReport{Valid: false, Length: n, FirstBrokenIndex: i, FirstBrokenReason: "height discontinuity"}
		}
		if len(b.Hash) == 0 {
			return &IntegrityReport{Valid: false, Length: n, FirstBrokenIndex: i, FirstBrokenReason: "empty hash"}
		}
		if i > 0 && !bytes.Equal(b.PrevHash, blocks[i-1].Hash) {
			return &IntegrityReport{Valid: false, Length: n, FirstBrokenIndex: i, FirstBrokenReason: "prev-hash chain break"}
		}
		if ok, reason := verifyBlockPoW(b); !ok {
			return &IntegrityReport{Valid: false, Length: n, FirstBrokenIndex: i, FirstBrokenReason: reason}
		}
	}
	return &IntegrityReport{Valid: true, Length: n, FirstBrokenIndex: -1}
}

// verifyBlockPoW confirms the block's stored hash is a valid proof-of-work over
// its fields (PrevHash + Data + nonce + difficulty): the recomputed digest must
// meet the target AND equal the stored hash.
func verifyBlockPoW(b *Block) (bool, string) {
	pow := NewProofOfWork(b)
	if !pow.Validate() {
		return false, "proof of work target not met"
	}
	digest := sha256.Sum256(pow.InitNonce(int(b.Nonce)))
	if !bytes.Equal(digest[:], b.Hash) {
		return false, "stored hash does not match recomputed proof"
	}
	return true, ""
}

func New() *Chain {
	return &Chain{Blocks: []*Block{Genesis()}}
}

// NewWithDB builds an independent persistent chain over db. It uses the same
// schema, reload verification and conflict handling as the process singleton,
// without claiming the global instance. It returns an error rather than a
// partial chain when persisted blocks cannot be decoded or verified. Callers
// that pair the chain with a domain repository must pass the same SQLite
// database so transactional block appends and business rows share one commit.
func NewWithDB(db *sql.DB) (*Chain, error) {
	chain := New()
	if err := bindChainToDB(chain, db); err != nil {
		return nil, err
	}
	return chain, nil
}
