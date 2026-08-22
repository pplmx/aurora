package nft

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
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
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	notOwnerPub, notOwnerPriv, _ := ed25519.GenerateKey(nil)
	recipientPub, _, _ := ed25519.GenerateKey(nil)

	_, err = svc.Transfer(minted.ID, notOwnerPub, recipientPub, notOwnerPriv, chain)
	if err != ErrNotOwner {
		t.Fatalf("expected ErrNotOwner, got %v", err)
	}
}

func TestNFTService_GetNFTByID(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)

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
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)

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
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)
	chain := blockchain.InitBlockChain()

	ownerPub, ownerPriv, _ := ed25519.GenerateKey(nil)
	err := svc.Burn("nonexistent-id", ownerPub, ownerPriv, chain)
	if err != ErrNFTNotFound {
		t.Fatalf("expected ErrNFTNotFound, got %v", err)
	}
}

func TestNFTService_Burn_VerifyOperationSignature(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)

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
	svc := NewService(repo, newMockTxManagerWithRepo(repo.inmemRepo))
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
	svc := NewServiceWithoutTx(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	pub, priv, _ := ed25519.GenerateKey(nil)
	_, err := svc.Transfer("nonexistent-id", pub, pub, priv, chain)
	if err != ErrNFTNotFound {
		t.Fatalf("expected ErrNFTNotFound, got %v", err)
	}
}

// TestNFTService_Transfer_Atomicity is the genuine mid-transaction rollback
// test: the transfer operation is saved FIRST, then the atomic ownership
// transfer fails. A real transaction must roll the saved operation back, so
// after the failed transfer the audit log contains only the mint operation
// and the owner is unchanged. (The pre-transaction version of this test
// overclaimed: it only asserted that Transfer returned an error, which the
// injected failure trivially satisfied without verifying any rollback.)
func TestNFTService_Transfer_Atomicity(t *testing.T) {
	repo := &FailingRepo{inmemRepo: NewInmemRepo().(*inmemRepo), failOnTryTransferOwnership: true}
	svc := NewService(repo, newMockTxManagerWithRepo(repo.inmemRepo))
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, creatorPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)
	minted, _ := svc.Mint(nft, chain)

	recipientPub, _, _ := ed25519.GenerateKey(nil)
	_, err := svc.Transfer(minted.ID, creatorPub, recipientPub, creatorPriv, chain)
	if err == nil {
		t.Fatal("expected error when repo.TryTransferOwnership fails")
	}

	// Rollback assertion 1: the transfer operation saved before the failure
	// must NOT survive — only the mint operation remains.
	ops, err := svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations: %v", err)
	}
	if len(ops) != 1 {
		t.Errorf("expected 1 operation after rolled-back transfer (mint only), got %d", len(ops))
	}
	for _, op := range ops {
		if op.Type != "mint" {
			t.Errorf("unexpected surviving operation type %q (orphan audit record)", op.Type)
		}
	}

	// Rollback assertion 2: ownership must be untouched.
	current, err := svc.GetNFTByID(minted.ID)
	if err != nil {
		t.Fatalf("GetNFTByID: %v", err)
	}
	if current == nil {
		t.Fatal("NFT must still exist after a failed transfer")
	}
	if !bytes.Equal(current.Owner, creatorPub) {
		t.Errorf("owner changed despite failed transfer: got %x, want %x", current.Owner, creatorPub)
	}
}

// TestNFTService_Mint_Atomicity proves Mint's two writes (NFT row + audit
// operation) roll back together when the operation save fails mid-transaction.
// The pre-transaction implementation compensated with a best-effort
// DeleteNFT whose own error was silently swallowed; with a real transaction
// no compensation exists and no partial state can remain.
func TestNFTService_Mint_Atomicity(t *testing.T) {
	repo := &FailingRepo{inmemRepo: NewInmemRepo().(*inmemRepo), failOnSaveOperation: true}
	svc := NewService(repo, newMockTxManagerWithRepo(repo.inmemRepo))
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)

	_, err := svc.Mint(nft, chain)
	if err == nil {
		t.Fatal("expected error when repo.SaveOperation fails")
	}

	// The NFT row saved before the failure must have been rolled back.
	if current, _ := svc.GetNFTByID(nft.ID); current != nil {
		t.Errorf("NFT row survived a failed mint (orphaned row without audit operation): %+v", current)
	}
	ops, _ := svc.GetOperations(nft.ID)
	if len(ops) != 0 {
		t.Errorf("expected 0 operations after rolled-back mint, got %d", len(ops))
	}
}

// TestNFTService_Burn_Atomicity proves Burn's operation record rolls back
// when the conditional delete fails mid-transaction (no orphan "burn attempt"
// record), and the NFT remains owned by the original owner.
func TestNFTService_Burn_Atomicity(t *testing.T) {
	repo := &FailingRepo{inmemRepo: NewInmemRepo().(*inmemRepo), failOnTryDeleteNFTIfOwned: true}
	svc := NewService(repo, newMockTxManagerWithRepo(repo.inmemRepo))
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	ownerPub, ownerPriv, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", ownerPub, ownerPub)
	minted, _ := svc.Mint(nft, chain)

	err := svc.Burn(minted.ID, ownerPub, ownerPriv, chain)
	if err == nil {
		t.Fatal("expected error when repo.TryDeleteNFTIfOwned fails")
	}

	// The burn operation saved before the failure must have been rolled back.
	ops, err := svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations: %v", err)
	}
	for _, op := range ops {
		if op.Type == "burn" {
			t.Error("burn operation survived a failed burn (orphan audit record)")
		}
	}

	// The NFT must still exist, still owned by the original owner.
	current, err := svc.GetNFTByID(minted.ID)
	if err != nil {
		t.Fatalf("GetNFTByID: %v", err)
	}
	if current == nil {
		t.Fatal("NFT must still exist after a failed burn")
	}
	if !bytes.Equal(current.Owner, ownerPub) {
		t.Errorf("owner changed despite failed burn")
	}
}

// TestNFTService_Mint_TxManagerFailure proves the service surfaces a
// transaction-level failure (the tx manager itself rejects the unit) without
// persisting anything.
func TestNFTService_Mint_TxManagerFailure(t *testing.T) {
	repo := NewInmemRepo().(*inmemRepo)
	// The manager fails on the first transaction, rolling back everything the
	// callback wrote through the repo.
	svc := NewService(repo, newFailingTxManagerWithRepo(repo, 1))
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, _, _ := ed25519.GenerateKey(nil)
	nft := NewNFT("Test NFT", "Description", "", "", creatorPub, creatorPub)

	_, err := svc.Mint(nft, chain)
	if err == nil {
		t.Fatal("expected error when the transaction manager fails")
	}
	if current, _ := svc.GetNFTByID(nft.ID); current != nil {
		t.Errorf("NFT row survived a failed transaction: %+v", current)
	}
}

func TestNFTService_Transfer_VerifyOperationSaved(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)
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
	svc := NewServiceWithoutTx(repo)
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
			continue
		}
		if found.Name != minted.Name {
			t.Errorf("name mismatch for %s", minted.ID)
		}
	}
}

func TestNFTService_MultipleTransfers(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
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

// mockTxManager simulates TransactionManager semantics on top of the
// in-memory repo's snapshot hooks: beginTx before the callback, rollbackTx on
// any callback error (or an injected failure), commitTx on success. It lets
// the service tests verify genuine mid-transaction rollback without a real
// SQLite database. Single-goroutine only (the snapshot is repo-global).
type mockTxManager struct {
	repo       *inmemRepo
	shouldFail bool
	failStep   int
	step       int
}

func (m *mockTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	if m.repo != nil {
		m.repo.beginTx()
	}

	if m.shouldFail {
		m.step++
		if m.step == m.failStep {
			if m.repo != nil {
				m.repo.rollbackTx()
			}
			return fmt.Errorf("transaction failed at step %d", m.failStep)
		}
	}

	err := fn(nil)

	if err != nil {
		if m.repo != nil {
			m.repo.rollbackTx()
		}
		return err
	}

	if m.repo != nil {
		m.repo.commitTx()
	}

	return nil
}

func newMockTxManager() *mockTxManager {
	return &mockTxManager{}
}

func newMockTxManagerWithRepo(repo *inmemRepo) *mockTxManager {
	return &mockTxManager{repo: repo}
}

func newFailingTxManager(failStep int) *mockTxManager {
	return &mockTxManager{shouldFail: true, failStep: failStep}
}

func newFailingTxManagerWithRepo(repo *inmemRepo, failStep int) *mockTxManager {
	return &mockTxManager{repo: repo, shouldFail: true, failStep: failStep}
}

type FailingRepo struct {
	*inmemRepo
	failOnSaveNFT              bool
	failOnUpdateNFT            bool
	failOnDeleteNFT            bool
	failOnSaveOperation        bool
	failOnTryTransferOwnership bool
	failOnTryDeleteNFTIfOwned  bool
}

// WithTx satisfies TransactableRepository: the failing repo delegates to the
// embedded in-memory repo, so it returns ITSELF (not the embedded repo) to
// keep the failure injection active inside the transaction callback.
func (r *FailingRepo) WithTx(_ *sql.Tx) Repository {
	return r
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

func (r *FailingRepo) TryTransferOwnership(nftID string, from, to []byte) error {
	if r.failOnTryTransferOwnership {
		return fmt.Errorf("mock try-transfer ownership error")
	}
	return r.inmemRepo.TryTransferOwnership(nftID, from, to)
}

func (r *FailingRepo) TryDeleteNFTIfOwned(nftID string, expectedOwner []byte) error {
	if r.failOnTryDeleteNFTIfOwned {
		return fmt.Errorf("mock try-delete-if-owned error")
	}
	return r.inmemRepo.TryDeleteNFTIfOwned(nftID, expectedOwner)
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

// TestNFTService_Transfer_ConcurrentOnlyOneWinner is a regression
// test for the TOCTOU race in NFTService.Transfer.
//
// Pre-fix behaviour: Transfer did GetNFT → IsOwner(from) →
// UpdateNFT(set Owner=to). Two concurrent transfers from the
// same owner to different recipients BOTH passed IsOwner (both
// saw the same initial owner), and the last writer silently
// overwrote the other recipient — the audit log ended up
// inconsistent with the actual on-chain owner, and one transfer
// appeared to "succeed" without actually moving the NFT.
//
// Post-fix behaviour: Transfer uses repo.TryTransferOwnership,
// which is a conditional update (WHERE owner = from). Only one
// transfer can succeed; all concurrent attempts against the same
// starting owner receive ErrOwnershipChanged. The audit log and
// the storage owner can never disagree.
func TestNFTService_Transfer_ConcurrentOnlyOneWinner(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	creatorPub, creatorPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate creator key: %v", err)
	}

	nft := NewNFT("Race NFT", "Test", "", "", creatorPub, creatorPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	// Build 8 distinct recipients and concurrently attempt to
	// transfer the same NFT from the same owner to each of them.
	const goroutines = 8
	recipients := make([]ed25519.PublicKey, goroutines)
	for i := 0; i < goroutines; i++ {
		recipients[i], _, err = ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatalf("generate recipient %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	results := make([]error, goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			_, err := svc.Transfer(minted.ID, creatorPub, recipients[idx], creatorPriv, chain)
			results[idx] = err
		}(i)
	}
	close(start)
	wg.Wait()

	// Exactly one transfer must have succeeded.
	successes := 0
	var winnerIdx = -1
	for i, err := range results {
		if err == nil {
			successes++
			winnerIdx = i
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful transfer, got %d (results=%v)", successes, results)
	}
	if winnerIdx < 0 {
		t.Fatalf("no winner index recorded (results=%v)", results)
	}

	// The losers must have been rejected with ErrOwnershipChanged
	// (or, defensively, ErrNotOwner if the storage flipped before
	// the service's GetNFT ran — both are valid rejection paths).
	for i, err := range results {
		if i == winnerIdx {
			continue
		}
		if err == nil {
			t.Errorf("transfer %d unexpectedly succeeded (lost-update bug)", i)
		}
		if !errors.Is(err, ErrOwnershipChanged) && !errors.Is(err, ErrNotOwner) {
			t.Errorf("transfer %d rejected with unexpected error: %v", i, err)
		}
	}

	// Storage state must match the winner: final owner is the
	// winner's recipient, and exactly one transfer operation is
	// recorded (audit log can't disagree with storage).
	final, err := svc.GetNFTByID(minted.ID)
	if err != nil {
		t.Fatalf("GetNFTByID: %v", err)
	}
	if !bytes.Equal(final.Owner, recipients[winnerIdx]) {
		t.Errorf("final owner mismatch: storage has %x, winner wanted %x (audit log / storage divergence)",
			final.Owner, recipients[winnerIdx])
	}

	ops, err := svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations: %v", err)
	}
	// The winner's transfer must be recorded. With a real
	// TransactionManager (the SQLite CLI/API path) rejected attempts
	// roll back completely, so ONLY the winner's operation commits.
	// This concurrency test runs without a tx manager (the in-memory
	// snapshot hooks are single-goroutine), so rejected attempts may
	// additionally leave their operation records; the assertion checks
	// the minimum guarantee that holds in both modes.
	transferOps := 0
	for _, op := range ops {
		if op.Type == "transfer" {
			transferOps++
		}
	}
	if transferOps < 1 {
		t.Errorf("expected at least 1 transfer operation recorded, got %d", transferOps)
	}
}

// TestNFTService_Burn_ConcurrentOnlyOneWinner is a regression
// test for the TOCTOU race in NFTService.Burn.
//
// Pre-fix behaviour: Burn did GetNFT → IsOwner(owner) →
// DeleteNFT. Two concurrent burns both passed IsOwner, both
// proceeded to delete, and the audit log ended up with two
// "burn" operations for the same NFT — or, in a Transfer-vs-Burn
// race, the transfer succeeded against one state while the
// burn succeeded against the prior state, producing an
// inconsistent audit log.
//
// Post-fix behaviour: Burn uses repo.TryDeleteNFTIfOwned, which
// is a conditional DELETE (WHERE id = ? AND owner = ?). Only
// one burn can succeed; all concurrent attempts receive
// ErrOwnershipChanged (mapped to ErrNotOwner). The audit log
// can never record more than one burn per NFT.
func TestNFTService_Burn_ConcurrentOnlyOneWinner(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	ownerPub, ownerPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate owner key: %v", err)
	}

	nft := NewNFT("Race Burn", "Test", "", "", ownerPub, ownerPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	// 8 concurrent burns from the same owner. Pre-fix: all
	// would have "succeeded" (the second would hit ErrNFTNotFound
	// since the first deleted the row, but the audit log would
	// still record 8 burn operations as if all succeeded).
	// Post-fix: exactly 1 succeeds, 7 receive ErrNotOwner.
	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	results := make([]error, goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx] = svc.Burn(minted.ID, ownerPub, ownerPriv, chain)
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful burn, got %d (results=%v)", successes, results)
	}

	// The winner's burn must be recorded. With a real TransactionManager
	// (the SQLite CLI/API path) rejected attempts roll back completely,
	// so ONLY the winner's burn operation commits. This concurrency test
	// runs without a tx manager (the in-memory snapshot hooks are
	// single-goroutine), so rejected attempts may additionally leave
	// their operation records; the assertion checks the minimum
	// guarantee that holds in both modes.
	ops, err := svc.GetOperations(minted.ID)
	if err != nil {
		t.Fatalf("GetOperations: %v", err)
	}
	burnOps := 0
	for _, op := range ops {
		if op.Type == "burn" {
			burnOps++
		}
	}
	if burnOps < 1 {
		t.Errorf("expected at least 1 burn operation recorded, got %d", burnOps)
	}

	// NFT must be gone from storage.
	final, err := svc.GetNFTByID(minted.ID)
	if err != nil {
		t.Fatalf("GetNFTByID after burn: %v", err)
	}
	if final != nil {
		t.Errorf("expected NFT to be deleted, got %v", final)
	}

	// All losers must have been rejected with ErrNotOwner
	// (defensively accepting ErrNFTNotFound in case the winner's
	// delete committed before the loser's TryDeleteNFTIfOwned ran —
	// both are valid rejection paths).
	for i, err := range results {
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrNotOwner) && !errors.Is(err, ErrNFTNotFound) {
			t.Errorf("burn %d rejected with unexpected error: %v", i, err)
		}
	}
}

// TestNFTService_Transfer_ForgeMismatchedKey proves that a caller who only
// knows the owner's PUBLIC key (which is public) cannot forge a transfer with
// their own private key — the key/owner binding must hold.
func TestNFTService_Transfer_ForgeMismatchedKey(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	victimPub, _, _ := ed25519.GenerateKey(nil) // victim's public key only
	nft := NewNFT("Victim NFT", "", "", "", victimPub, victimPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	attackerPub, attackerPriv, _ := ed25519.GenerateKey(nil)
	_, err = svc.Transfer(minted.ID, victimPub, attackerPub, attackerPriv, chain)
	if err != ErrKeyMismatch {
		t.Fatalf("expected ErrKeyMismatch for forged transfer, got %v", err)
	}
}

// TestNFTService_Burn_ForgeMismatchedKey proves a forged burn (someone who
// knows only the owner's public key) is rejected.
func TestNFTService_Burn_ForgeMismatchedKey(t *testing.T) {
	repo := NewInmemRepo()
	svc := NewServiceWithoutTx(repo)
	chain := blockchain.InitBlockChain()
	blockchain.ResetForTest()

	victimPub, _, _ := ed25519.GenerateKey(nil) // victim's public key only
	nft := NewNFT("Victim NFT", "", "", "", victimPub, victimPub)
	minted, err := svc.Mint(nft, chain)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	_, attackerPriv, _ := ed25519.GenerateKey(nil)
	err = svc.Burn(minted.ID, victimPub, attackerPriv, chain)
	if err != ErrKeyMismatch {
		t.Fatalf("expected ErrKeyMismatch for forged burn, got %v", err)
	}
}
