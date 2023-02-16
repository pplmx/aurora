package models

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// Block represents a single block in the blockchain
type Block struct {
	Index     int    // position of the block in the chain
	Timestamp string // time when the block was created
	Data      string // data stored in the block (e.g. transactions)
	Hash      string // hash of the block header
	PrevHash  string // hash of the previous block
}

// CreateBlock creates a new block using the given data
func CreateBlock(prevBlock Block, data string) Block {
	var block Block

	block.Index = prevBlock.Index + 1
	block.Timestamp = time.Now().String()
	block.Data = data
	block.PrevHash = prevBlock.Hash
	block.Hash = CalculateHash(block)

	return block
}

// CalculateHash returns the hexadecimal hash of a block's header (index, timestamp, data, prevHash)
func CalculateHash(block Block) string {
	header := strconv.Itoa(block.Index) + block.Timestamp + block.Data + block.PrevHash

	hash := sha256.New()
	hash.Write([]byte(header))
	hashed := hash.Sum(nil)

	return hex.EncodeToString(hashed)
}

// IsBlockValid checks if a new candidate block is valid by comparing its index and previous hash with those of the last block in the chain
func IsBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if CalculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}
