// Package blockchain provides core blockchain infrastructure including
// block, blockchain, and proof-of-work implementations.
package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"sync"
	"time"
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
}

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
// The whole critical section — read latest, compute height, append —
// happens under the write lock. NewBlock's expensive PoW computation
// happens *outside* the lock (see below) so that concurrent AddBlock
// calls can run PoW in parallel. The lock only protects the slice
// itself, which is what was racing.
func (c *BlockChain) AddBlock(data string) (int64, error) {
	if c == nil {
		return 0, fmt.Errorf("blockchain not initialized")
	}

	// Take the read lock just long enough to copy the previous hash and
	// the current length. Released before PoW so multiple AddBlock
	// callers can mine in parallel.
	c.mu.RLock()
	if len(c.Blocks) == 0 {
		c.mu.RUnlock()
		return 0, fmt.Errorf("blockchain not initialized")
	}
	prevHash := append([]byte(nil), c.Blocks[len(c.Blocks)-1].Hash...)
	height := int64(len(c.Blocks))
	c.mu.RUnlock()

	newBlock := NewBlock(data, prevHash)
	newBlock.Height = height

	c.mu.Lock()
	c.Blocks = append(c.Blocks, newBlock)
	appendedAt := int64(len(c.Blocks) - 1)
	c.mu.Unlock()

	return appendedAt, nil
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

func NewBlockChain() *BlockChain {
	return &BlockChain{Blocks: []*Block{Genesis()}}
}
