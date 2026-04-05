package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
)

type Block struct {
	Height    int64
	Hash      []byte
	PrevHash  []byte
	Data      []byte
	Nonce     int64
	Timestamp int64
}

type BlockChain struct {
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
		Timestamp: 0,
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

func (c *BlockChain) AddBlock(data string) int64 {
	prevBlock := c.Blocks[len(c.Blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	newBlock.Height = int64(len(c.Blocks))
	c.Blocks = append(c.Blocks, newBlock)
	height := len(c.Blocks) - 1

	return int64(height)
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

func (c *BlockChain) GetBlockData(blockHeight int64) (string, error) {
	if blockHeight < 0 || blockHeight >= int64(len(c.Blocks)) {
		return "", fmt.Errorf("invalid block height: %d", blockHeight)
	}
	return string(c.Blocks[blockHeight].Data), nil
}

func (c *BlockChain) GetLotteryRecords() []string {
	records := make([]string, 0)
	for _, block := range c.Blocks {
		data := string(block.Data)
		if len(data) > 0 && data != "Genesis" {
			records = append(records, data)
		}
	}
	return records
}

func NewBlockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}
