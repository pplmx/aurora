package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
)

type BlockChain struct {
	Blocks []*Block
}

type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
	Nonce    int
}

func (b *Block) DeriveHash() {
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
	// This will join our previous block's relevant info with the new blocks
	hash := sha256.Sum256(info)
	//This performs the actual hashing algorithm
	b.Hash = hash[:]
	//If this ^ doesn't make sense, you can look up slice defaults
}

func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	// Don't forget to add the 0 at the end for the nonce!
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func (chain *BlockChain) AddBlock(data string) int64 {
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	newBlock := CreateBlock(data, prevBlock.Hash)
	chain.Blocks = append(chain.Blocks, newBlock)
	return int64(len(chain.Blocks) - 1)
}

func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func InitBlockChain() *BlockChain {
	// Try to load from SQLite first
	chain, err := LoadFromDB()
	if err == nil && len(chain.Blocks) > 1 {
		return chain
	}
	// Fall back to in-memory if DB doesn't exist or is empty
	return &BlockChain{[]*Block{Genesis()}}
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)

	Handle(err)

	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)

	Handle(err)

	return &block
}

func (chain *BlockChain) AddLotteryRecord(data string) (int64, error) {
	chain.AddBlock(data)
	return int64(len(chain.Blocks) - 1), nil
}

func (chain *BlockChain) GetBlockData(blockHeight int64) (string, error) {
	if blockHeight < 0 || blockHeight >= int64(len(chain.Blocks)) {
		return "", fmt.Errorf("invalid block height: %d", blockHeight)
	}
	return string(chain.Blocks[blockHeight].Data), nil
}

func (chain *BlockChain) GetLotteryRecords() []string {
	records := make([]string, 0)
	for _, block := range chain.Blocks {
		data := string(block.Data)
		if len(data) > 0 && data != "Genesis" {
			records = append(records, data)
		}
	}
	return records
}
