package blockchain

import (
	"context"
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
	nonce, hash, err := pow.Run(context.Background())
	if err != nil {
		t.Fatalf("mine: %v", err)
	}

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

func TestProofOfWorkRun_Success(t *testing.T) {
	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("test data"),
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()
	pow := NewProofOfWork(block)

	nonce, hash, err := pow.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestProofOfWorkRun_Interrupt(t *testing.T) {
	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("interrupt test"),
		Timestamp: time.Now().Unix(),
	}

	pow := NewProofOfWork(block)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := pow.Run(ctx)
	if err == nil {
		t.Fatal("expected error on context cancellation, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestProofOfWorkRun_Deadline(t *testing.T) {
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

	_, _, err := pow.Run(ctx)
	if err == nil {
		t.Fatal("expected error on deadline exceeded, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

// TestProofOfWorkValidate pins the proof check the chain actually runs:
// BlockChain.Verify calls pow.Validate(), and the standalone ValidateProof
// helper that duplicated its comparison was removed.
func TestProofOfWorkValidate(t *testing.T) {
	block := &Block{
		Height:    0,
		PrevHash:  []byte{},
		Data:      []byte("validation test"),
		Timestamp: time.Now().Unix(),
	}

	pow := NewProofOfWork(block)

	nonce, _, err := pow.Run(context.Background())
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	block.Nonce = int64(nonce)
	if !pow.Validate() {
		t.Error("a freshly mined proof must validate")
	}

	block.Nonce = int64(nonce + 1)
	if pow.Validate() {
		t.Error("a tampered nonce must not validate")
	}
}
