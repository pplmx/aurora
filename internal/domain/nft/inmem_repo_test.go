package nft

import (
	"testing"
)

// TestInmemRepo_WithTx verifies the tx-scoped repository returned by WithTx
// writes through to the same underlying store as the base repository. The
// in-memory implementation intentionally returns itself unchanged; the test
// pins that a write via the tx repo is visible via the base repo.
func TestInmemRepo_WithTx(t *testing.T) {
	repo := NewInmemRepo()
	txRepo := repo.WithTx(nil)
	if txRepo == nil {
		t.Fatal("WithTx must return a non-nil Repository")
	}

	if err := txRepo.SaveNFT(&NFT{ID: "n1", Name: "A", Owner: []byte("owner")}); err != nil {
		t.Fatalf("SaveNFT via WithTx repo: %v", err)
	}
	got, err := repo.GetNFT("n1")
	if err != nil {
		t.Fatalf("GetNFT: %v", err)
	}
	if got == nil {
		t.Fatal("expected NFT written via WithTx repo to be visible via base repo")
	}
}

// TestInmemRepo_UpdateNFT proves UpdateNFT replaces the stored record (the
// field used by the non-atomic alias of Transfer).
func TestInmemRepo_UpdateNFT(t *testing.T) {
	repo := NewInmemRepo()
	if err := repo.SaveNFT(&NFT{ID: "n1", Name: "A", Owner: []byte("owner1")}); err != nil {
		t.Fatalf("SaveNFT: %v", err)
	}

	replacement := &NFT{ID: "n1", Name: "B", Owner: []byte("owner2")}
	if err := repo.UpdateNFT(replacement); err != nil {
		t.Fatalf("UpdateNFT: %v", err)
	}

	got, err := repo.GetNFT("n1")
	if err != nil {
		t.Fatalf("GetNFT: %v", err)
	}
	if got == nil {
		t.Fatal("expected NFT to exist after UpdateNFT")
	}
	if got.Name != "B" || string(got.Owner) != "owner2" {
		t.Errorf("UpdateNFT did not replace record: got name=%q owner=%q", got.Name, got.Owner)
	}
}

// TestInmemRepo_DeleteNFT proves DeleteNFT removes the record.
func TestInmemRepo_DeleteNFT(t *testing.T) {
	repo := NewInmemRepo()
	if err := repo.SaveNFT(&NFT{ID: "n1", Name: "A", Owner: []byte("owner")}); err != nil {
		t.Fatalf("SaveNFT: %v", err)
	}
	if err := repo.DeleteNFT("n1"); err != nil {
		t.Fatalf("DeleteNFT: %v", err)
	}

	got, err := repo.GetNFT("n1")
	if err != nil {
		t.Fatalf("GetNFT: %v", err)
	}
	if got != nil {
		t.Errorf("expected NFT to be deleted, still present: %+v", got)
	}
}
