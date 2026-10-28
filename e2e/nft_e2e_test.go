package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"sort"
	"testing"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/infra/blockchain"
)

type inMemoryNFTRepo struct {
	nfts        map[string]*nft.NFT
	operations  map[string][]*nft.Operation
	nftsByOwner map[string][]*nft.NFT
}

func newInMemoryNFTRepo() *inMemoryNFTRepo {
	return &inMemoryNFTRepo{
		nfts:        make(map[string]*nft.NFT),
		operations:  make(map[string][]*nft.Operation),
		nftsByOwner: make(map[string][]*nft.NFT),
	}
}

func (r *inMemoryNFTRepo) SaveNFT(nft *nft.NFT) error {
	r.nfts[nft.ID] = nft
	ownerKey := string(nft.Owner)
	r.nftsByOwner[ownerKey] = append(r.nftsByOwner[ownerKey], nft)
	return nil
}

func (r *inMemoryNFTRepo) GetNFT(id string) (*nft.NFT, error) {
	return r.nfts[id], nil
}

func (r *inMemoryNFTRepo) ListNFTs(limit, offset int) ([]*nft.NFT, error) {
	var all []*nft.NFT
	for _, byOwner := range r.nftsByOwner {
		all = append(all, byOwner...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	if limit > 0 {
		if offset < 0 {
			offset = 0
		}
		if offset >= len(all) {
			return nil, nil
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		all = all[offset:end]
	}
	return all, nil
}

func (r *inMemoryNFTRepo) GetNFTsByOwner(owner []byte, limit, offset int) ([]*nft.NFT, error) {
	all := r.nftsByOwner[string(owner)]
	if limit > 0 {
		if offset < 0 {
			offset = 0
		}
		if offset >= len(all) {
			return nil, nil
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		all = all[offset:end]
	}
	return all, nil
}

func (r *inMemoryNFTRepo) UpdateNFT(item *nft.NFT) error {
	old, ok := r.nfts[item.ID]
	if !ok {
		return nft.ErrNFTNotFound
	}
	oldOwnerKey := string(old.Owner)
	newOwnerKey := string(item.Owner)

	var updatedList []*nft.NFT
	for _, n := range r.nftsByOwner[oldOwnerKey] {
		if n.ID != item.ID {
			updatedList = append(updatedList, n)
		}
	}
	r.nftsByOwner[oldOwnerKey] = updatedList

	r.nfts[item.ID] = item
	r.nftsByOwner[newOwnerKey] = append(r.nftsByOwner[newOwnerKey], item)
	return nil
}

func (r *inMemoryNFTRepo) TryTransferOwnership(nftID string, from, to []byte) error {
	existing, ok := r.nfts[nftID]
	if !ok {
		return nft.ErrNFTNotFound
	}
	if string(existing.Owner) != string(from) {
		return nft.ErrOwnershipChanged
	}
	oldOwnerKey := string(existing.Owner)
	newOwnerKey := string(to)
	var updatedList []*nft.NFT
	for _, n := range r.nftsByOwner[oldOwnerKey] {
		if n.ID != nftID {
			updatedList = append(updatedList, n)
		}
	}
	r.nftsByOwner[oldOwnerKey] = updatedList
	existing.Owner = to
	r.nfts[nftID] = existing
	r.nftsByOwner[newOwnerKey] = append(r.nftsByOwner[newOwnerKey], existing)
	return nil
}

func (r *inMemoryNFTRepo) DeleteNFT(id string) error {
	item, ok := r.nfts[id]
	if !ok {
		return nft.ErrNFTNotFound
	}
	ownerKey := string(item.Owner)
	for i, n := range r.nftsByOwner[ownerKey] {
		if n.ID == id {
			r.nftsByOwner[ownerKey] = append(r.nftsByOwner[ownerKey][:i], r.nftsByOwner[ownerKey][i+1:]...)
			break
		}
	}
	delete(r.nfts, id)
	return nil
}

func (r *inMemoryNFTRepo) TryDeleteNFTIfOwned(nftID string, expectedOwner []byte) error {
	existing, ok := r.nfts[nftID]
	if !ok {
		return nft.ErrNFTNotFound
	}
	if string(existing.Owner) != string(expectedOwner) {
		return nft.ErrOwnershipChanged
	}
	ownerKey := string(existing.Owner)
	for i, n := range r.nftsByOwner[ownerKey] {
		if n.ID == nftID {
			r.nftsByOwner[ownerKey] = append(r.nftsByOwner[ownerKey][:i], r.nftsByOwner[ownerKey][i+1:]...)
			break
		}
	}
	delete(r.nfts, nftID)
	return nil
}

func (r *inMemoryNFTRepo) SaveOperation(op *nft.Operation) error {
	r.operations[op.NFTID] = append(r.operations[op.NFTID], op)
	return nil
}

func (r *inMemoryNFTRepo) GetOperations(nftID string, limit, offset int) ([]*nft.Operation, error) {
	operations := r.operations[nftID]
	if offset >= len(operations) {
		return nil, nil
	}
	operations = operations[offset:]
	if limit > 0 && len(operations) > limit {
		operations = operations[:limit]
	}
	return operations, nil
}

// WithTx satisfies nft.TransactableRepository. E2E tests run
// single-goroutine against this in-memory repo with no transaction manager,
// so the repo simply returns itself (same approach as inMemoryTokenRepo).
func (r *inMemoryNFTRepo) WithTx(_ *sql.Tx) nft.Repository {
	return r
}

func TestNFTE2E(t *testing.T) {
	repo := newInMemoryNFTRepo()
	service := nft.NewServiceWithoutTx(repo)
	chain := blockchain.Init()

	_, creatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	creatorPub := creatorPriv.Public().(ed25519.PublicKey)

	_, toPriv, _ := ed25519.GenerateKey(rand.Reader)
	toPub := toPriv.Public().(ed25519.PublicKey)

	nftItem := &nft.NFT{
		Name:        "Test NFT",
		Description: "Description",
		ImageURL:    "https://example.com/img.png",
		TokenURI:    "",
		Owner:       creatorPub,
		Creator:     creatorPub,
	}

	minted, err := service.Mint(nftItem, chain)
	if err != nil {
		t.Fatal(err)
	}

	if minted.Name != "Test NFT" {
		t.Errorf("Name = %v, want Test NFT", minted.Name)
	}

	op, err := service.Transfer(minted.ID, creatorPub, toPub, creatorPriv, chain)
	if err != nil {
		t.Fatal(err)
	}
	if op.Type != "transfer" {
		t.Errorf("Operation = %v, want transfer", op.Type)
	}

	updated, _ := service.GetNFTByID(minted.ID)
	if string(updated.Owner) == "" {
		t.Error("Owner should be updated")
	}

	ops, _ := service.GetOperations(minted.ID, 0, 0)
	if len(ops) != 2 {
		t.Errorf("len(ops) = %v, want 2 (mint + transfer)", len(ops))
	}

	err = service.Burn(minted.ID, toPub, toPriv, chain)
	if err != nil {
		t.Fatal(err)
	}

	deleted, _ := service.GetNFTByID(minted.ID)
	if deleted != nil {
		t.Error("NFT should be deleted")
	}

	t.Log("E2E test passed!")
}

func TestNFTMultipleOwners(t *testing.T) {
	repo := newInMemoryNFTRepo()
	service := nft.NewServiceWithoutTx(repo)
	chain := blockchain.Init()

	_, user1Priv, _ := ed25519.GenerateKey(rand.Reader)
	user1Pub := user1Priv.Public().(ed25519.PublicKey)

	_, user2Priv, _ := ed25519.GenerateKey(rand.Reader)
	user2Pub := user2Priv.Public().(ed25519.PublicKey)

	_, user3Priv, _ := ed25519.GenerateKey(rand.Reader)
	user3Pub := user3Priv.Public().(ed25519.PublicKey)

	nft1 := &nft.NFT{Name: "NFT 1", Owner: user1Pub, Creator: user1Pub}
	nft2 := &nft.NFT{Name: "NFT 2", Owner: user1Pub, Creator: user1Pub}
	nft3 := &nft.NFT{Name: "NFT 3", Owner: user1Pub, Creator: user1Pub}

	minted1, _ := service.Mint(nft1, chain)
	minted2, _ := service.Mint(nft2, chain)
	_, _ = minted1, minted2
	_, _ = service.Mint(nft3, chain)

	_, _ = service.Transfer(minted1.ID, user1Pub, user2Pub, user1Priv, chain)
	_, _ = service.Transfer(minted2.ID, user1Pub, user3Pub, user1Priv, chain)

	user2NFTs, _ := service.GetNFTsByOwner(user2Pub, 0, 0)
	user3NFTs, _ := service.GetNFTsByOwner(user3Pub, 0, 0)

	if len(user2NFTs) != 1 {
		t.Errorf("user2 NFTs = %v, want 1", len(user2NFTs))
	}
	if len(user3NFTs) != 1 {
		t.Errorf("user3 NFTs = %v, want 1", len(user3NFTs))
	}

	t.Log("Multi-owner test passed!")
}
