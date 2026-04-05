package blockchain

import (
	"testing"
)

func TestBlock_DeriveHash(t *testing.T) {
	block := &Block{
		Data:     []byte("test data"),
		PrevHash: []byte("previous hash"),
		Nonce:    0,
	}

	block.DeriveHash()

	if len(block.Hash) == 0 {
		t.Error("Hash should not be empty")
	}
}

func TestCreateBlock(t *testing.T) {
	data := "test block data"
	prevHash := []byte("previous hash")

	block := CreateBlock(data, prevHash)

	if block == nil {
		t.Fatal("CreateBlock returned nil")
	}
	if len(block.Hash) == 0 {
		t.Error("Block hash should not be empty")
	}
	if string(block.Data) != data {
		t.Errorf("Data = %v, want %v", string(block.Data), data)
	}
}

func TestGenesis(t *testing.T) {
	block := Genesis()

	if block == nil {
		t.Fatal("Genesis returned nil")
	}
	if string(block.Data) != "Genesis" {
		t.Errorf("Genesis data = %v, want 'Genesis'", string(block.Data))
	}
	if len(block.Hash) == 0 {
		t.Error("Genesis hash should not be empty")
	}
}

func TestBlockChain_AddBlock(t *testing.T) {
	ResetForTest()
	chain := &BlockChain{[]*Block{Genesis()}}

	height := chain.AddBlock("test data")

	if height != 1 {
		t.Errorf("AddBlock returned height = %v, want 1", height)
	}
	if len(chain.Blocks) != 2 {
		t.Errorf("Blocks len = %v, want 2", len(chain.Blocks))
	}
}

func TestBlockChain_AddLotteryRecord(t *testing.T) {
	ResetForTest()
	chain := &BlockChain{[]*Block{Genesis()}}

	height, err := chain.AddLotteryRecord("lottery: test")

	if err != nil {
		t.Fatalf("AddLotteryRecord failed: %v", err)
	}
	if height != 1 {
		t.Errorf("Height = %v, want 1", height)
	}
}

func TestBlockChain_GetBlockData(t *testing.T) {
	ResetForTest()
	chain := &BlockChain{[]*Block{Genesis()}}
	chain.AddBlock("test data")

	data, err := chain.GetBlockData(1)
	if err != nil {
		t.Fatalf("GetBlockData failed: %v", err)
	}
	if data != "test data" {
		t.Errorf("Data = %v, want 'test data'", data)
	}
}

func TestBlockChain_GetBlockData_InvalidHeight(t *testing.T) {
	chain := &BlockChain{[]*Block{Genesis()}}

	_, err := chain.GetBlockData(-1)
	if err == nil {
		t.Error("Expected error for negative height")
	}

	_, err = chain.GetBlockData(100)
	if err == nil {
		t.Error("Expected error for height out of range")
	}
}

func TestBlockChain_GetLotteryRecords(t *testing.T) {
	ResetForTest()
	chain := &BlockChain{[]*Block{Genesis()}}
	chain.AddBlock("lottery 1")
	chain.AddBlock("lottery 2")
	chain.AddBlock("Genesis")

	records := chain.GetLotteryRecords()

	if len(records) != 2 {
		t.Errorf("GetLotteryRecords returned %v records, want 2", len(records))
	}
}

func TestBlock_Serialize(t *testing.T) {
	block := &Block{
		Hash:     []byte("hash"),
		Data:     []byte("data"),
		PrevHash: []byte("prev"),
		Nonce:    42,
	}

	data := block.Serialize()
	if len(data) == 0 {
		t.Error("Serialized data should not be empty")
	}
}

func TestDeserialize(t *testing.T) {
	block := &Block{
		Hash:     []byte("hash"),
		Data:     []byte("data"),
		PrevHash: []byte("prev"),
		Nonce:    42,
	}

	data := block.Serialize()
	deserialized := Deserialize(data)

	if string(deserialized.Data) != string(block.Data) {
		t.Errorf("Deserialized data = %v, want %v", string(deserialized.Data), string(block.Data))
	}
	if deserialized.Nonce != block.Nonce {
		t.Errorf("Nonce = %v, want %v", deserialized.Nonce, block.Nonce)
	}
}
