package nft

import (
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

func (r *inmemRepo) DeleteNFT(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nfts, id)
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
