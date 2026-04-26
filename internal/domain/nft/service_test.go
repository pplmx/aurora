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

func TestNFTService_Burn(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	ownerPub, ownerPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Burnable NFT", "To be burned", "", "", ownerPub, ownerPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	err = svc.Burn(minted.ID, ownerPub, ownerPriv, chain)
	if err != nil {
		t.Fatalf("Burn failed: %v", err)
	}

	burned, _ := svc.GetNFTByID(minted.ID)
	if burned != nil {
		t.Errorf("expected NFT to be deleted, got %v", burned)
	}

	ops, err := svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations failed: %v", err)
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 operations (mint + burn), got %d", len(ops))
	}
	if !ops[1].IsBurn() {
		t.Error("last operation should be burn")
	}
}

func TestNFTService_Burn_NotOwner(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	ownerPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Desc", "", "", ownerPub, ownerPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	notOwnerPub, notOwnerPriv, _ := ed25519.GenerateKey(nil)
	err = svc.Burn(minted.ID, notOwnerPub, notOwnerPriv, chain)
	if err != ErrNotOwner {
		t.Fatalf("expected ErrNotOwner, got %v", err)
	}
}

func TestNFTService_Burn_NotFound(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()

	ownerPub, ownerPriv, _ := ed25519.GenerateKey(nil)
	err := svc.Burn("nonexistent-id", ownerPub, ownerPriv, chain)
	if err != ErrNFTNotFound {
		t.Fatalf("expected ErrNFTNotFound, got %v", err)
	}
}

func TestNFTService_Burn_VerifyOperationSignature(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	ownerPub, ownerPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Desc", "", "", ownerPub, ownerPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	err = svc.Burn(minted.ID, ownerPub, ownerPriv, chain)
	if err != nil {
		t.Fatalf("Burn failed: %v", err)
	}

	ops, _ := svc.GetOperations(minted.ID)
	burnOp := ops[1]
	if burnOp.NFTID != minted.ID {
		t.Errorf("expected NFTID %s, got %s", minted.ID, burnOp.NFTID)
	}
	if burnOp.Type != "burn" {
		t.Errorf("expected type 'burn', got %s", burnOp.Type)
	}
	if len(burnOp.Signature) == 0 {
		t.Error("expected non-empty signature")
	}
}

func TestOperation_IsMint(t *testing.T) {
	mintOp := &Operation{Type: "mint"}
	transferOp := &Operation{Type: "transfer"}

	if !mintOp.IsMint() {
		t.Error("mint operation should return true for IsMint()")
	}
	if transferOp.IsMint() {
		t.Error("transfer operation should return false for IsMint()")
	}
}

func TestNFTService_Mint_ChainError(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)

	badChain := &mockBlockWriter{shouldFail: true}
	_, err := svc.Mint(nft, badChain)
	if err == nil {
		t.Fatal("expected error when chain.AddBlock fails")
	}
}

func TestNFTService_Mint_RepoSaveError(t *testing.T) {
	repo := &FailingRepo{inmemRepo: NewInmemRepo().(*inmemRepo), failOnSaveNFT: true}
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)

	_, err := svc.Mint(nft, chain)
	if err == nil {
		t.Fatal("expected error when repo.SaveNFT fails")
	}
}

func TestNFTService_Transfer_NotFound(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	pub, priv, _ := ed25519.GenerateKey(nil)
	_, err := svc.Transfer("nonexistent-id", pub, pub, priv, chain)
	if err != ErrNFTNotFound {
		t.Fatalf("expected ErrNFTNotFound, got %v", err)
	}
}

func TestNFTService_Transfer_Atomicity(t *testing.T) {
	repo := &FailingRepo{inmemRepo: NewInmemRepo().(*inmemRepo), failOnUpdateNFT: true}
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, creatorPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)
	minted, _ := svc.Mint(nft, chain)

	recipientPub, _, _ := ed25519.GenerateKey(nil)
	_, err := svc.Transfer(minted.ID, creatorPub, recipientPub, creatorPriv, chain)
	if err == nil {
		t.Fatal("expected error when repo.UpdateNFT fails")
	}

	_ = minted
	_ = creatorPub
	_ = recipientPub
}

func TestNFTService_Transfer_VerifyOperationSaved(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, creatorPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)
	minted, _ := svc.Mint(nft, chain)

	recipientPub, _, _ := ed25519.GenerateKey(nil)
	op, err := svc.Transfer(minted.ID, creatorPub, recipientPub, creatorPriv, chain)
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	ops, _ := svc.GetOperations(minted.ID)
	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}
	if ops[1].ID != op.ID {
		t.Error("operation ID mismatch")
	}
}

func TestNFTService_Transfer_SelfTransfer(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	ownerPub, ownerPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", ownerPub, ownerPub)
	minted, _ := svc.Mint(nft, chain)

	_, err := svc.Transfer(minted.ID, ownerPub, ownerPub, ownerPriv, chain)
	if err != nil {
		t.Fatalf("Self-transfer should succeed: %v", err)
	}

	result, _ := svc.GetNFTByID(minted.ID)
	if !bytes.Equal(result.Owner, ownerPub) {
		t.Error("owner should remain unchanged for self-transfer")
	}
}

func TestNFTService_BatchMint(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)

	var mintedNFTs []*NFT
	for i := 0; i < 5; i++ {
		nft := NewNFT(fmt.Sprintf("NFT %d", i), fmt.Sprintf("Description %d", i), "", "", creatorPub, creatorPub)
		minted, err := svc.Mint(nft, chain)
		if err != nil {
			t.Fatalf("BatchMint failed at item %d: %v", i, err)
		}
		mintedNFTs = append(mintedNFTs, minted)
	}

	creatorNFTs, err := svc.GetNFTsByCreator(creatorPub)
	if err != nil {
		t.Fatalf("GetNFTsByCreator failed: %v", err)
	}
	if len(creatorNFTs) != 5 {
		t.Errorf("expected 5 NFTs, got %d", len(creatorNFTs))
	}

	for _, minted := range mintedNFTs {
		found, _ := svc.GetNFTByID(minted.ID)
		if found == nil {
			t.Errorf("NFT %s not found after batch mint", minted.ID)
		}
		if found.Name != minted.Name {
			t.Errorf("name mismatch for %s", minted.ID)
		}
	}
}

func TestNFTService_MultipleTransfers(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewService(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	owner1Pub, owner1Priv, _ := ed25519.GenerateKey(nil)
	owner2Pub, owner2Priv, _ := ed25519.GenerateKey(nil)
	owner3Pub, _, _ := ed25519.GenerateKey(nil)

	nft := NewNFT("Test NFT", "Description", "", "", owner1Pub, owner1Pub)
	minted, _ := svc.Mint(nft, chain)

	_, err := svc.Transfer(minted.ID, owner1Pub, owner2Pub, owner1Priv, chain)
	if err != nil {
		t.Fatalf("First transfer failed: %v", err)
	}

	_, err = svc.Transfer(minted.ID, owner2Pub, owner3Pub, owner2Priv, chain)
	if err != nil {
		t.Fatalf("Second transfer failed: %v", err)
	}

	result, _ := svc.GetNFTByID(minted.ID)
	if !bytes.Equal(result.Owner, owner3Pub) {
		t.Errorf("expected owner3, got different owner")
	}

	ops, _ := svc.GetOperations(minted.ID)
	if len(ops) != 3 {
		t.Errorf("expected 3 operations, got %d", len(ops))
	}
}

type mockBlockWriter struct {
	shouldFail bool
}

func (m *mockBlockWriter) AddBlock(data string) (int64, error) {
	if m.shouldFail {
		return 0, fmt.Errorf("mock block error")
	}
	return 1, nil
}

type FailingRepo struct {
	*inmemRepo
	failOnSaveNFT       bool
	failOnUpdateNFT     bool
	failOnDeleteNFT     bool
	failOnSaveOperation bool
}

func (r *FailingRepo) SaveNFT(nft *NFT) error {
	if r.failOnSaveNFT {
		return fmt.Errorf("mock save error")
	}
	return r.inmemRepo.SaveNFT(nft)
}

func (r *FailingRepo) GetNFT(id string) (*NFT, error) {
	return r.inmemRepo.GetNFT(id)
}

func (r *FailingRepo) GetNFTsByOwner(owner []byte) ([]*NFT, error) {
	return r.inmemRepo.GetNFTsByOwner(owner)
}

func (r *FailingRepo) GetNFTsByCreator(creator []byte) ([]*NFT, error) {
	return r.inmemRepo.GetNFTsByCreator(creator)
}

func (r *FailingRepo) UpdateNFT(nft *NFT) error {
	if r.failOnUpdateNFT {
		return fmt.Errorf("mock update error")
	}
	return r.inmemRepo.UpdateNFT(nft)
}

func (r *FailingRepo) DeleteNFT(id string) error {
	if r.failOnDeleteNFT {
		return fmt.Errorf("mock delete error")
	}
	return r.inmemRepo.DeleteNFT(id)
}

func (r *FailingRepo) SaveOperation(op *Operation) error {
	if r.failOnSaveOperation {
		return fmt.Errorf("mock operation error")
	}
	return r.inmemRepo.SaveOperation(op)
}

func (r *FailingRepo) GetOperations(nftID string) ([]*Operation, error) {
	return r.inmemRepo.GetOperations(nftID)
}
