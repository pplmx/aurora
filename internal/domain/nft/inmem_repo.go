package nft

import (
	"bytes"
	"sync"
)

type inmemRepo struct {
	mu         sync.RWMutex
	nfts       map[string]*NFT
	operations map[string][]*Operation
}

func NewInmemRepo() Repository {
	return &inmemRepo{
		nfts:       make(map[string]*NFT),
		operations: make(map[string][]*Operation),
	}
}

func (r *inmemRepo) SaveNFT(nft *NFT) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nfts[nft.ID] = nft
	return nil
}

func (r *inmemRepo) GetNFT(id string) (*NFT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nfts[id], nil
}

func (r *inmemRepo) GetNFTsByOwner(owner []byte) ([]*NFT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*NFT
	for _, nft := range r.nfts {
		if nft.IsOwner(owner) {
			result = append(result, nft)
		}
	}
	return result, nil
}

func (r *inmemRepo) GetNFTsByCreator(creator []byte) ([]*NFT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*NFT
	for _, nft := range r.nfts {
		if len(nft.Creator) == len(creator) {
			match := true
			for i := range nft.Creator {
				if nft.Creator[i] != creator[i] {
					match = false
					break
				}
			}
			if match {
				result = append(result, nft)
			}
		}
	}
	return result, nil
}

func (r *inmemRepo) UpdateNFT(nft *NFT) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nfts[nft.ID] = nft
	return nil
}

// TryTransferOwnership mirrors the SQLite primitive: under the
// single repo-wide write lock, it checks that the current owner
// still matches `from` and atomically writes the new owner. The
// lock guarantees no other goroutine can read+write between our
// check and our store, so the read-modify-write window that the
// pre-fix NFTService.Transfer had is closed.
func (r *inmemRepo) TryTransferOwnership(nftID string, from, to []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.nfts[nftID]
	if !ok {
		return ErrNFTNotFound
	}
	if !bytes.Equal(existing.Owner, from) {
		return ErrOwnershipChanged
	}
	existing.Owner = to
	return nil
}

func (r *inmemRepo) DeleteNFT(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nfts, id)
	return nil
}

// TryDeleteNFTIfOwned mirrors the SQLite primitive: under the
// single repo-wide write lock, it checks that the current owner
// still matches `expectedOwner` and atomically deletes the NFT.
// The lock guarantees no concurrent TryTransferOwnership (or
// another TryDeleteNFTIfOwned) can sneak in between the check
// and the delete.
func (r *inmemRepo) TryDeleteNFTIfOwned(nftID string, expectedOwner []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.nfts[nftID]
	if !ok {
		return ErrNFTNotFound
	}
	if !bytes.Equal(existing.Owner, expectedOwner) {
		return ErrOwnershipChanged
	}
	delete(r.nfts, nftID)
	return nil
}

func (r *inmemRepo) SaveOperation(op *Operation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operations[op.NFTID] = append(r.operations[op.NFTID], op)
	return nil
}

func (r *inmemRepo) GetOperations(nftID string) ([]*Operation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.operations[nftID], nil
}
