package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/nft"
)

func TestNFTE2E(t *testing.T) {
	storage := nft.NewNFTStorage()
	nft.SetNFTStorage(storage)
	chain := blockchain.InitBlockChain()

	_, creatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	creatorPub := creatorPriv.Public().(ed25519.PublicKey)

	_, toPriv, _ := ed25519.GenerateKey(rand.Reader)
	toPub := toPriv.Public().(ed25519.PublicKey)

	nftItem, err := nft.MintNFT("Test NFT", "Description", "https://example.com/img.png", "", creatorPub, chain)
	if err != nil {
		t.Fatal(err)
	}

	if nftItem.Name != "Test NFT" {
		t.Errorf("Name = %v, want Test NFT", nftItem.Name)
	}

	op, err := nft.TransferNFT(nftItem.ID, creatorPub, creatorPriv, toPub, chain)
	if err != nil {
		t.Fatal(err)
	}
	if op.Operation != "transfer" {
		t.Errorf("Operation = %v, want transfer", op.Operation)
	}

	updated, _ := nft.GetNFTByID(nftItem.ID)
	if updated.Owner == "" {
		t.Error("Owner should be updated")
	}

	ops, _ := nft.GetNFTOperations(nftItem.ID)
	if len(ops) != 2 {
		t.Errorf("len(ops) = %v, want 2 (mint + transfer)", len(ops))
	}

	err = nft.BurnNFT(nftItem.ID, toPub, toPriv, chain)
	if err != nil {
		t.Fatal(err)
	}

	deleted, _ := nft.GetNFTByID(nftItem.ID)
	if deleted != nil {
		t.Error("NFT should be deleted")
	}

	t.Log("E2E test passed!")
}

func TestNFTMultipleOwners(t *testing.T) {
	storage := nft.NewNFTStorage()
	nft.SetNFTStorage(storage)
	chain := blockchain.InitBlockChain()

	_, user1Priv, _ := ed25519.GenerateKey(rand.Reader)
	user1Pub := user1Priv.Public().(ed25519.PublicKey)

	_, user2Priv, _ := ed25519.GenerateKey(rand.Reader)
	user2Pub := user2Priv.Public().(ed25519.PublicKey)

	_, user3Priv, _ := ed25519.GenerateKey(rand.Reader)
	user3Pub := user3Priv.Public().(ed25519.PublicKey)

	nft1, _ := nft.MintNFT("NFT 1", "", "", "", user1Pub, chain)
	nft2, _ := nft.MintNFT("NFT 2", "", "", "", user1Pub, chain)
	nft3, _ := nft.MintNFT("NFT 3", "", "", "", user1Pub, chain)

	nft.TransferNFT(nft1.ID, user1Pub, user1Priv, user2Pub, chain)
	nft.TransferNFT(nft2.ID, user1Pub, user1Priv, user3Pub, chain)
	_ = nft3

	user1NFTs, _ := nft.GetNFTsByOwner(user1Pub)
	user2NFTs, _ := nft.GetNFTsByOwner(user2Pub)
	user3NFTs, _ := nft.GetNFTsByOwner(user3Pub)

	if len(user1NFTs) != 1 {
		t.Errorf("user1 NFTs = %v, want 1", len(user1NFTs))
	}
	if len(user2NFTs) != 1 {
		t.Errorf("user2 NFTs = %v, want 1", len(user2NFTs))
	}
	if len(user3NFTs) != 1 {
		t.Errorf("user3 NFTs = %v, want 1", len(user3NFTs))
	}

	t.Log("Multi-owner test passed!")
}
