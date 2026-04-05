package nft

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func TestNFTService_VerifyTransfer_Valid(t *testing.T) {
	service := &NFTService{}

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	toPub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	timestamp := time.Now().Unix()
	nftID := "nft-123"
	fromB64 := base64.StdEncoding.EncodeToString(pub)
	toB64 := base64.StdEncoding.EncodeToString(toPub)
	message := fmt.Sprintf("%s|%s|%s|%d", nftID, fromB64, toB64, timestamp)

	messageHash := sha256.Sum256([]byte(message))
	signature := ed25519.Sign(priv, messageHash[:])

	op := &Operation{
		ID:        "op-1",
		NFTID:     nftID,
		Type:      "transfer",
		From:      pub,
		To:        toPub,
		Signature: signature,
		Timestamp: timestamp,
	}

	valid, err := service.VerifyTransfer(op)
	if err != nil {
		t.Fatalf("VerifyTransfer failed: %v", err)
	}

	if !valid {
		t.Error("Expected valid transfer signature")
	}
}

func TestNFTService_VerifyTransfer_NotTransfer(t *testing.T) {
	service := &NFTService{}

	op := &Operation{
		ID:    "op-1",
		NFTID: "nft-123",
		Type:  "mint",
	}

	valid, err := service.VerifyTransfer(op)
	if err != nil {
		t.Fatalf("VerifyTransfer failed: %v", err)
	}

	if valid {
		t.Error("Expected false for non-transfer operation")
	}
}

func TestNFTService_VerifyTransfer_InvalidSignature(t *testing.T) {
	service := &NFTService{}

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	toPub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	timestamp := time.Now().Unix()
	nftID := "nft-123"
	fromB64 := base64.StdEncoding.EncodeToString(pub)
	toB64 := base64.StdEncoding.EncodeToString(toPub)
	message := fmt.Sprintf("%s|%s|%s|%d", nftID, fromB64, toB64, timestamp)

	messageHash := sha256.Sum256([]byte(message))
	_, wrongPriv, _ := ed25519.GenerateKey(nil)
	signature := ed25519.Sign(wrongPriv, messageHash[:])

	op := &Operation{
		ID:        "op-1",
		NFTID:     nftID,
		Type:      "transfer",
		From:      pub,
		To:        toPub,
		Signature: signature,
		Timestamp: timestamp,
	}

	valid, err := service.VerifyTransfer(op)
	if err != nil {
		t.Fatalf("VerifyTransfer failed: %v", err)
	}

	if valid {
		t.Error("Expected invalid signature")
	}
}
