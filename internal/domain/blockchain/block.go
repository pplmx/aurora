// Package blockchain provides core blockchain infrastructure including
// block, blockchain, and proof-of-work implementations.
package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"
	"time"

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

// BlockChain represents the blockchain data structure.
//
// Concurrency: Blocks is mutated only under mu. Reads (GetBlockData,
// GetLotteryRecords) also take mu (RLock) so HTTP handlers can read
// safely while other goroutines are appending blocks via AddBlock.
// Without this mutex, a real production server with concurrent API
// traffic lost blocks and tripped the race detector (see
// TestBlockChain_ConcurrentAddBlock_DataRace for the regression).
type BlockChain struct {
	mu     sync.RWMutex
	Blocks []*Block
	// persist, when non-nil, is invoked after a block is appended so the
	// chain can survive process restarts. It is left nil by NewBlockChain
	// (pure in-memory use, e.g. tests) and wired to the blocks table by
	// InitBlockChain. The in-memory slice stays authoritative; a persistence
	// failure is logged rather than fatal, matching the package's existing
	// "operate in non-persistent mode" fallback.
	//
	// The persist hook must not silently overwrite a committed block: when the
	// backing store already holds a conflicting block at the same height (a
	// second process sharing the DB file reserved it first), persist must
	// return ErrHeightConflict so appendBlock can drop the losing candidate and
	// retry at the DB-authoritative next height (TASK-244, ISS-242).
	persist func(*Block) error

	// syncFromDB, when non-nil, is invoked under the write lock *before* the
	// next height is reserved. It brings the in-memory chain up to date with
	// blocks a concurrent process committed to the shared DB file, so the
	// reserved height is DB-authoritative (never a stale len(c.Blocks)).
	// Left nil by NewBlockChain; wired by InitBlockChain.
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

func (b *Block) DeriveHash() {
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
	hash := sha256.Sum256(info)
	b.Hash = hash[:]
}

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
	nonce, hash := pow.Run()

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
func (c *BlockChain) AddBlock(data string) (int64, error) {
	return c.appendBlock(data, nil)
}

// appendBlock serializes the whole reserve+mine+append under the write lock
// (see AddBlock's comment for why mining cannot be parallelized). When stamp
// is non-nil it is called with the block height just before the block is
// mined, so callers can finalize the payload (e.g. stamp a self-describing
// height into an audit record's JSON) before Data becomes part of the PoW
// hash. A stamp that returns "" leaves data unchanged.
func (c *BlockChain) appendBlock(data string, stamp func(height int64) string) (int64, error) {
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

	for attempt := 0; attempt <= maxHeightRetries; attempt++ {
		// Re-sync from the shared DB before reserving, so the height we pick is
		// the DB-authoritative next height even when a concurrent process has
		// appended blocks this process has not seen yet. Failure degrades to
		// the stale in-memory height (best-effort, matching the non-persistent
		// fallback); the persist conflict check below is the safety net.
		if c.syncFromDB != nil {
			if err := c.syncFromDB(); err != nil {
				logger.Warn().Err(err).Msg("Failed to sync chain from database; using in-memory height")
			}
		}

		if len(c.Blocks) == 0 {
			return 0, fmt.Errorf("blockchain not initialized")
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
		// Mine under the lock: consecutive blocks cannot be mined in parallel
		// (see the AddBlock comment for why), so serializing is both correct
		// and the only valid option.
		pow := NewProofOfWork(block)
		nonce, hash := pow.Run()
		block.Nonce = int64(nonce)
		block.Hash = append([]byte(nil), hash...)
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

// ResetBlocks re-seeds the in-memory chain back to a single genesis block.
// It is used by `lottery reset` after the persisted `blocks` and
// `lottery_records` tables have been cleared: the chain singleton is created
// once (init.go's once.Do) and otherwise keeps the pre-reset blocks in memory,
// so a post-reset AddBlock would compute a stale height and a PrevHash that
// references a block the reset deleted. Re-seeding makes the next AddBlock
// start at height 1 with a valid genesis PrevHash (TASK-071, ISS-063).
func (c *BlockChain) ResetBlocks() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Blocks = []*Block{Genesis()}
}

func (b *Block) Serialize() ([]byte, error) {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)
	if err != nil {
		return nil, err
	}

	return res.Bytes(), nil
}

func Deserialize(data []byte) (*Block, error) {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)
	if err != nil {
		return nil, err
	}

	return &block, nil
}

func (c *BlockChain) GetBlockData(blockHeight int64) (string, error) {
	if c == nil {
		return "", fmt.Errorf("blockchain not initialized")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if blockHeight < 0 || blockHeight >= int64(len(c.Blocks)) {
		return "", fmt.Errorf("invalid block height: %d", blockHeight)
	}
	return string(c.Blocks[blockHeight].Data), nil
}

func (c *BlockChain) GetLotteryRecords() []string {
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

func (c *BlockChain) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Blocks)
}

// IntegrityReport summarises an integrity verification of the chain.
type IntegrityReport struct {
	Valid             bool   `json:"valid"`
	Length            int    `json:"length"`
	FirstBrokenIndex  int    `json:"first_broken_index,omitempty"` // -1 when valid
	FirstBrokenReason string `json:"first_broken_reason,omitempty"`
}

// VerifyIntegrity cryptographically verifies the whole chain: every block must
// carry a non-empty hash, each block's PrevHash must link to the previous
// block's hash, and each block's stored hash must be a valid proof-of-work over
// its fields. It returns a report with the first broken index (-1 if valid).
//
// This is the tamper-evidence surface behind the "blockchain-based" guarantees
// of the module (v1.25): operators can prove the ledger is internally consistent.
func (c *BlockChain) VerifyIntegrity() *IntegrityReport {
	if c == nil {
		return &IntegrityReport{Valid: false, FirstBrokenIndex: 0, FirstBrokenReason: "nil chain"}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	blocks := c.Blocks
	n := len(blocks)
	for i, b := range blocks {
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

func NewBlockChain() *BlockChain {
	return &BlockChain{Blocks: []*Block{Genesis()}}
}
