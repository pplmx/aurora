package nft

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/blockchain"
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

func TestNFTService_Transfer(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, creatorPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	recipientPub, _, _ := ed25519.GenerateKey(nil)
	_, err = svc.Transfer(minted.ID, creatorPub, recipientPub, creatorPriv, chain)
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	transferred, err := svc.GetNFTByID(minted.ID)
	if err != nil {
		t.Fatalf("GetNFTByID failed: %v", err)
	}
	if !bytes.Equal(transferred.Owner, recipientPub) {
		t.Errorf("expected owner to be recipient, got %v", transferred.Owner)
	}

	ops, err := svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations failed: %v", err)
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 operations (mint + transfer), got %d", len(ops))
	}
}

func TestNFTService_Transfer_NotOwner(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	_, notOwnerPub, _ := ed25519.GenerateKey(nil)
	_, wrongPriv, _ := ed25519.GenerateKey(nil)
	recipientPub, _, _ := ed25519.GenerateKey(nil)

	_, err = svc.Transfer(minted.ID, notOwnerPub, recipientPub, wrongPriv, chain)
	if err != ErrNotOwner {
		t.Fatalf("expected ErrNotOwner, got %v", err)
	}
}

func TestNFTService_GetNFTByID(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	found, err := svc.GetNFTByID(minted.ID)
	if err != nil {
		t.Fatalf("GetNFTByID failed: %v", err)
	}
	if found.ID != minted.ID {
		t.Errorf("expected NFT ID %s, got %s", minted.ID, found.ID)
	}
	if found.Name != "Test NFT" {
		t.Errorf("expected NFT name 'Test NFT', got %s", found.Name)
	}
}

func TestNFTService_GetNFTByID_NotFound(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	nft, err := svc.GetNFTByID("nonexistent")
	if nft != nil {
		t.Errorf("expected nil NFT, got %v", nft)
	}
	if err != nil {
		t.Errorf("expected no error for not found, got %v", err)
	}
}

func TestNFTService_GetNFTsByOwner(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	ownerPub, _, _ := ed25519.GenerateKey(nil)
	otherPub, _, _ := ed25519.GenerateKey(nil)

	nft1 := NewNFT("NFT 1", "Desc", "", "", ownerPub, ownerPub)
	nft2 := NewNFT("NFT 2", "Desc", "", "", ownerPub, ownerPub)
	nft3 := NewNFT("NFT 3", "Desc", "", "", otherPub, otherPub)

	_, err := svc.Mint(nft1, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}
	_, err = svc.Mint(nft2, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}
	_, err = svc.Mint(nft3, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	ownerNFTs, err := svc.GetNFTsByOwner(ownerPub)
	if err != nil {
		t.Fatalf("GetNFTsByOwner failed: %v", err)
	}
	if len(ownerNFTs) != 2 {
		t.Errorf("expected 2 NFTs for owner, got %d", len(ownerNFTs))
	}

	otherNFTs, err := svc.GetNFTsByOwner(otherPub)
	if err != nil {
		t.Fatalf("GetNFTsByOwner failed: %v", err)
	}
	if len(otherNFTs) != 1 {
		t.Errorf("expected 1 NFT for other, got %d", len(otherNFTs))
	}
}

func TestNFTService_GetNFTsByCreator(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	otherPub, _, _ := ed25519.GenerateKey(nil)

	nft1 := NewNFT("NFT 1", "Desc", "", "", creatorPub, creatorPub)
	nft2 := NewNFT("NFT 2", "Desc", "", "", creatorPub, creatorPub)
	nft3 := NewNFT("NFT 3", "Desc", "", "", otherPub, otherPub)

	_, err := svc.Mint(nft1, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}
	_, err = svc.Mint(nft2, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}
	_, err = svc.Mint(nft3, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	creatorNFTs, err := svc.GetNFTsByCreator(creatorPub)
	if err != nil {
		t.Fatalf("GetNFTsByCreator failed: %v", err)
	}
	if len(creatorNFTs) != 2 {
		t.Errorf("expected 2 NFTs for creator, got %d", len(creatorNFTs))
	}

	otherNFTs, err := svc.GetNFTsByCreator(otherPub)
	if err != nil {
		t.Fatalf("GetNFTsByCreator failed: %v", err)
	}
	if len(otherNFTs) != 1 {
		t.Errorf("expected 1 NFT for other, got %d", len(otherNFTs))
	}
}

func TestNFTService_GetNFTsByOwner_Empty(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	pub, _, _ := ed25519.GenerateKey(nil)
	nfts, err := svc.GetNFTsByOwner(pub)
	if err != nil {
		t.Fatalf("GetNFTsByOwner failed: %v", err)
	}
	if len(nfts) != 0 {
		t.Errorf("expected 0 NFTs, got %d", len(nfts))
	}
}
