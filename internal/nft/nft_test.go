package nft

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
)

func TestMintNFT(t *testing.T) {
	storage := NewNFTStorage()
	SetNFTStorage(storage)
	chain := blockchain.InitBlockChain()

	_, pub, _ := ed25519.GenerateKey(rand.Reader)

	nft, err := MintNFT("Test NFT", "Description", "https://example.com/img.png", "", pub, chain)
	if err != nil {
		t.Fatal(err)
	}

	if nft.Name != "Test NFT" {
		t.Errorf("Name = %v, want Test NFT", nft.Name)
	}

	stored, _ := storage.GetNFT(nft.ID)
	if stored == nil {
		t.Error("NFT should be stored")
	}
}

func TestTransferNFT(t *testing.T) {
	storage := NewNFTStorage()
	SetNFTStorage(storage)
	chain := blockchain.InitBlockChain()

	_, creatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	creatorPub := creatorPriv.Public().(ed25519.PublicKey)

	_, toPriv, _ := ed25519.GenerateKey(rand.Reader)
	toPub := toPriv.Public().(ed25519.PublicKey)

	nft, _ := MintNFT("Test NFT", "Desc", "", "", creatorPub, chain)

	op, err := TransferNFT(nft.ID, creatorPub, creatorPriv, toPub, chain)
	if err != nil {
		t.Fatal(err)
	}

	if op.Operation != "transfer" {
		t.Errorf("Operation = %v, want transfer", op.Operation)
	}

	if !VerifyTransfer(op) {
		t.Error("Transfer verification should pass")
	}
}

func TestBurnNFT(t *testing.T) {
	storage := NewNFTStorage()
	SetNFTStorage(storage)
	chain := blockchain.InitBlockChain()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pub := priv.Public().(ed25519.PublicKey)

	nft, _ := MintNFT("Test NFT", "Desc", "", "", pub, chain)

	err := BurnNFT(nft.ID, pub, priv, chain)
	if err != nil {
		t.Fatal(err)
	}

	deleted, _ := storage.GetNFT(nft.ID)
	if deleted != nil {
		t.Error("NFT should be deleted")
	}
}
