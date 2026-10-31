// Building and resetting the chain: block construction with its proof of work,
// the genesis block, the in-memory reset, and the two constructors that decide
// whether the chain is backed by the shared database.
package blockchain

import (
	"context"
	"database/sql"
	"time"
)

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
// Head returns the newest block's height and hash. The hash is copied so a
// caller cannot alias the chain's storage; an empty chain reports (-1, nil).
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
