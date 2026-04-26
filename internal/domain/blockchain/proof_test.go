package blockchain

import (
	"context"
	"math/big"
	"testing"
	"time"
)

func TestProofOfWork_Run(t *testing.T) {
	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("test data"),
		Timestamp: time.Now().Unix(),
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	if nonce < 0 {
		t.Errorf("nonce should be non-negative, got %d", nonce)
	}

	if len(hash) != 32 {
		t.Errorf("hash should be 32 bytes, got %d", len(hash))
	}

	pow.Block.Nonce = int64(nonce)
	if !pow.Validate() {
		t.Error("proof of work should be valid")
	}
}

func TestMineBlockWithContext_Success(t *testing.T) {
	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("test data"),
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()
	pow := NewProofOfWork(block)

	nonce, hash, err := MineBlockWithContext(ctx, pow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if nonce < 0 {
		t.Errorf("nonce should be non-negative, got %d", nonce)
	}

	if len(hash) != 32 {
		t.Errorf("hash should be 32 bytes, got %d", len(hash))
	}

	data := pow.InitNonce(nonce)
	if !ValidateProof(data, hash, pow.Target) {
		t.Error("proof of work should be valid")
	}
}

func TestMineBlockWithContext_Interrupt(t *testing.T) {
	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("interrupt test"),
		Timestamp: time.Now().Unix(),
	}

	pow := NewProofOfWork(block)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := MineBlockWithContext(ctx, pow)
	if err == nil {
		t.Fatal("expected error on context cancellation, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestMineBlockWithContext_Deadline(t *testing.T) {
	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("deadline test"),
		Timestamp: time.Now().Unix(),
	}

	pow := NewProofOfWork(block)

	past := time.Now().Add(-1 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), past)
	defer cancel()

	_, _, err := MineBlockWithContext(ctx, pow)
	if err == nil {
		t.Fatal("expected error on deadline exceeded, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestValidateProof(t *testing.T) {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-Difficulty))

	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("validation test"),
		Timestamp: time.Now().Unix(),
	}

	pow := NewProofOfWork(block)

	nonce, _ := pow.Run()
	data := pow.InitNonce(nonce)

	if !ValidateProof(data, nil, target) {
		t.Error("valid proof should pass validation")
	}

	invalidData := pow.InitNonce(nonce + 1)
	if ValidateProof(invalidData, nil, target) {
		t.Error("invalid proof should fail validation")
	}
}
