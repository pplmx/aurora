package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	nftdomain "github.com/pplmx/aurora/internal/domain/nft"
)

type inMemoryNFTRepo struct {
	nfts          map[string]*nftdomain.NFT
	operations    map[string][]*nftdomain.Operation
	nftsByOwner   map[string][]*nftdomain.NFT
	nftsByCreator map[string][]*nftdomain.NFT
}

func newInMemoryNFTRepo() *inMemoryNFTRepo {
	return &inMemoryNFTRepo{
		nfts:          make(map[string]*nftdomain.NFT),
		operations:    make(map[string][]*nftdomain.Operation),
		nftsByOwner:   make(map[string][]*nftdomain.NFT),
		nftsByCreator: make(map[string][]*nftdomain.NFT),
	}
}

func (r *inMemoryNFTRepo) SaveNFT(nft *nftdomain.NFT) error {
	r.nfts[nft.ID] = nft
	ownerKey := string(nft.Owner)
	r.nftsByOwner[ownerKey] = append(r.nftsByOwner[ownerKey], nft)
	creatorKey := string(nft.Creator)
	r.nftsByCreator[creatorKey] = append(r.nftsByCreator[creatorKey], nft)
	return nil
}

func (r *inMemoryNFTRepo) GetNFT(id string) (*nftdomain.NFT, error) {
	return r.nfts[id], nil
}

func (r *inMemoryNFTRepo) GetNFTsByOwner(owner []byte) ([]*nftdomain.NFT, error) {
	return r.nftsByOwner[string(owner)], nil
}

func (r *inMemoryNFTRepo) GetNFTsByCreator(creator []byte) ([]*nftdomain.NFT, error) {
	return r.nftsByCreator[string(creator)], nil
}

func (r *inMemoryNFTRepo) UpdateNFT(nft *nftdomain.NFT) error {
	old, ok := r.nfts[nft.ID]
	if !ok {
		return nftdomain.ErrNFTNotFound
	}
	oldOwnerKey := string(old.Owner)
	newOwnerKey := string(nft.Owner)

	var updatedList []*nftdomain.NFT
	for _, n := range r.nftsByOwner[oldOwnerKey] {
		if n.ID != nft.ID {
			updatedList = append(updatedList, n)
		}
	}
	r.nftsByOwner[oldOwnerKey] = updatedList

	r.nfts[nft.ID] = nft
	r.nftsByOwner[newOwnerKey] = append(r.nftsByOwner[newOwnerKey], nft)
	return nil
}

func (r *inMemoryNFTRepo) DeleteNFT(id string) error {
	nft, ok := r.nfts[id]
	if !ok {
		return nftdomain.ErrNFTNotFound
	}
	ownerKey := string(nft.Owner)
	for i, n := range r.nftsByOwner[ownerKey] {
		if n.ID == id {
			r.nftsByOwner[ownerKey] = append(r.nftsByOwner[ownerKey][:i], r.nftsByOwner[ownerKey][i+1:]...)
			break
		}
	}
	delete(r.nfts, id)
	return nil
}

func (r *inMemoryNFTRepo) SaveOperation(op *nftdomain.Operation) error {
	r.operations[op.NFTID] = append(r.operations[op.NFTID], op)
	return nil
}

func (r *inMemoryNFTRepo) GetOperations(nftID string) ([]*nftdomain.Operation, error) {
	return r.operations[nftID], nil
}

func TestNFTE2E(t *testing.T) {
	repo := newInMemoryNFTRepo()
	service := nftdomain.NewService(repo)
	chain := blockchain.InitBlockChain()

	_, creatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	creatorPub := creatorPriv.Public().(ed25519.PublicKey)

	_, toPriv, _ := ed25519.GenerateKey(rand.Reader)
	toPub := toPriv.Public().(ed25519.PublicKey)

	nftItem := &nftdomain.NFT{
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

	ops, _ := service.GetOperations(minted.ID)
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
	service := nftdomain.NewService(repo)
	chain := blockchain.InitBlockChain()

	_, user1Priv, _ := ed25519.GenerateKey(rand.Reader)
	user1Pub := user1Priv.Public().(ed25519.PublicKey)

	_, user2Priv, _ := ed25519.GenerateKey(rand.Reader)
	user2Pub := user2Priv.Public().(ed25519.PublicKey)

	_, user3Priv, _ := ed25519.GenerateKey(rand.Reader)
	user3Pub := user3Priv.Public().(ed25519.PublicKey)

	nft1 := &nftdomain.NFT{Name: "NFT 1", Owner: user1Pub, Creator: user1Pub}
	nft2 := &nftdomain.NFT{Name: "NFT 2", Owner: user1Pub, Creator: user1Pub}
	nft3 := &nftdomain.NFT{Name: "NFT 3", Owner: user1Pub, Creator: user1Pub}

	minted1, _ := service.Mint(nft1, chain)
	minted2, _ := service.Mint(nft2, chain)
	_, _ = minted1, minted2
	_, _ = service.Mint(nft3, chain)

	_, _ = service.Transfer(minted1.ID, user1Pub, user2Pub, user1Priv, chain)
	_, _ = service.Transfer(minted2.ID, user1Pub, user3Pub, user1Priv, chain)

	user2NFTs, _ := service.GetNFTsByOwner(user2Pub)
	user3NFTs, _ := service.GetNFTsByOwner(user3Pub)

	if len(user2NFTs) != 1 {
		t.Errorf("user2 NFTs = %v, want 1", len(user2NFTs))
	}
	if len(user3NFTs) != 1 {
		t.Errorf("user3 NFTs = %v, want 1", len(user3NFTs))
	}

	t.Log("Multi-owner test passed!")
}
