package nft

import (
	"bytes"
	"database/sql"
	"sync"
)

type inmemRepo struct {
	mu         sync.RWMutex
	nfts       map[string]*NFT
	operations map[string][]*Operation
	// snapshot holds the state captured by beginTx so rollbackTx can restore
	// it. nil when no (mock) transaction is open.
	snapshot *txSnapshot
}

// txSnapshot is a deep copy of the repo state taken at beginTx.
type txSnapshot struct {
	nfts       map[string]*NFT
	operations map[string][]*Operation
}

func NewInmemRepo() TransactableRepository {
	return &inmemRepo{
		nfts:       make(map[string]*NFT),
		operations: make(map[string][]*Operation),
	}
}

// WithTx satisfies TransactableRepository. The in-memory repo has no real
// transaction handle; it returns itself unchanged. Rollback semantics for
// tests are provided by the beginTx/rollbackTx/commitTx snapshot hooks,
// driven by the mock TransactionManager in service_test.go.
func (r *inmemRepo) WithTx(_ *sql.Tx) Repository {
	return r
}

func (r *inmemRepo) beginTx() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot = &txSnapshot{
		nfts:       make(map[string]*NFT, len(r.nfts)),
		operations: make(map[string][]*Operation, len(r.operations)),
	}
	for id, n := range r.nfts {
		clone := *n
		r.snapshot.nfts[id] = &clone
	}
	for id, ops := range r.operations {
		clone := make([]*Operation, len(ops))
		for i, op := range ops {
			opClone := *op
			clone[i] = &opClone
		}
		r.snapshot.operations[id] = clone
	}
}

func (r *inmemRepo) rollbackTx() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.snapshot == nil {
		return
	}
	r.nfts = r.snapshot.nfts
	r.operations = r.snapshot.operations
	r.snapshot = nil
}

func (r *inmemRepo) commitTx() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot = nil
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

func (r *inmemRepo) GetNFTsByOwner(owner []byte, limit, offset int) ([]*NFT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*NFT
	for _, nft := range r.nfts {
		if nft.IsOwner(owner) {
			result = append(result, nft)
		}
	}
	// Mirror the SQLite LIMIT/OFFSET semantics: 0,0 (limit <= 0) is
	// unbounded and returns the whole (ordered) collection (TASK-101,
	// ISS-093). Map iteration order is nondeterministic, so in-memory
	// paging is best-effort — only the SQLite repo carries a stable
	// rowid order.
	if limit > 0 {
		if offset < 0 {
			offset = 0
		}
		if offset >= len(result) {
			return nil, nil
		}
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	}
	return result, nil
}

func (r *inmemRepo) GetNFTsByCreator(creator []byte) ([]*NFT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*NFT
	for _, nft := range r.nfts {
		if bytes.Equal(nft.Creator, creator) {
			result = append(result, nft)
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
