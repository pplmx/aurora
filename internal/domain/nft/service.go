package nft

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pplmx/aurora/internal/domain/blockchain"
)

type Service interface {
	Mint(nft *NFT, chain blockchain.BlockWriter) (*NFT, error)
	Transfer(nftID string, from, to, privateKey []byte, chain blockchain.BlockWriter) (*Operation, error)
	Burn(nftID string, owner, privateKey []byte, chain blockchain.BlockWriter) error
	VerifyTransfer(op *Operation) (bool, error)
	GetNFTByID(id string) (*NFT, error)
	GetNFTsByOwner(ownerPub []byte, limit, offset int) ([]*NFT, error)
	GetNFTsByCreator(creatorPub []byte) ([]*NFT, error)
	GetOperations(nftID string) ([]*Operation, error)
}

type NFTService struct {
	repo      TransactableRepository
	txManager TransactionManager
}

func NewService(repo TransactableRepository, txManager TransactionManager) *NFTService {
	if txManager == nil {
		txManager = noOpTxManager{}
	}
	return &NFTService{repo: repo, txManager: txManager}
}

// noOpTxManager executes the callback directly without a transaction. Used by
// callers that build the service over a repository with no transaction
// support (in-memory repos in the TUI and tests).
type noOpTxManager struct{}

func (noOpTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

// NewServiceWithoutTx builds a service whose multi-step writes run directly
// against the repository with no enclosing transaction. Appropriate only for
// repositories without transaction support; the SQLite-backed service must
// use NewService with a real TransactionManager.
func NewServiceWithoutTx(repo TransactableRepository) *NFTService {
	return NewService(repo, noOpTxManager{})
}

// txRepo returns the repository scoped to the given transaction when one is
// active, otherwise the service's default repository. Every state mutation
// performed inside a WithTransaction callback MUST go through txRepo so the
// writes share the same SQLite transaction and roll back together.
func (s *NFTService) txRepo(tx *sql.Tx) Repository {
	if tx == nil {
		return s.repo
	}
	return s.repo.WithTx(tx)
}

func (s *NFTService) Mint(nft *NFT, chain blockchain.BlockWriter) (*NFT, error) {
	nft.ID = uuid.New().String()
	nft.Timestamp = time.Now().Unix()

	data := fmt.Sprintf("%s|%s|%s", nft.ID, nft.Name, nft.Owner)
	height, err := chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	nft.BlockHeight = height

	// Persist the NFT row and its mint operation as ONE transaction. The
	// pre-transaction implementation saved the NFT first and, if the
	// operation save failed, ran a best-effort DeleteNFT compensation whose
	// own error was silently swallowed (_ = repo.DeleteNFT) — a crash or a
	// failed compensation left an orphaned NFT row with no audit record.
	// With a real transaction the partial state can never commit.
	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		txRepo := s.txRepo(tx)
		if err := txRepo.SaveNFT(nft); err != nil {
			return err
		}
		op := NewOperation(nft.ID, "mint", nil, nft.Owner, nil)
		op.BlockHeight = height
		if err := txRepo.SaveOperation(op); err != nil {
			return fmt.Errorf("failed to save mint operation: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return nft, nil
}

// verifyOwnerKey ensures the presented private key actually corresponds to the
// claimed public key (from/owner). Without this check a caller who only knows
// a victim's public key (which is public) could sign with their own key and
// forge a transfer or burn as the victim — the atomic ownership check alone
// only proves the *public key* owns the NFT, not that this caller holds its
// private counterpart.
func verifyOwnerKey(pub, priv []byte) error {
	if len(priv) != ed25519.PrivateKeySize {
		return ErrInvalidPrivateKey
	}
	if len(pub) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}
	pubFromPriv, ok := ed25519.PrivateKey(priv).Public().(ed25519.PublicKey)
	if !ok || !bytes.Equal(pubFromPriv, pub) {
		return ErrKeyMismatch
	}
	return nil
}

func (s *NFTService) Transfer(nftID string, from, to, privateKey []byte, chain blockchain.BlockWriter) (*Operation, error) {
	// Existence check only — we deliberately do NOT call
	// nft.IsOwner(from) here. That would read nft.Owner outside
	// any lock, racing with another goroutine's
	// TryTransferOwnership write (race detected in
	// TestNFTService_Transfer_ConcurrentOnlyOneWinner under
	// -race: read at entity.go:43 / write at inmem_repo.go:93).
	// The atomic primitive below is the single source of truth
	// for ownership — it rejects with ErrOwnershipChanged if
	// ownership moved under us, which we map to ErrNotOwner to
	// preserve the public error contract.
	nft, err := s.repo.GetNFT(nftID)
	if err != nil {
		return nil, err
	}
	if nft == nil {
		return nil, ErrNFTNotFound
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPrivateKey
	}
	if len(from) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	if len(to) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	// The caller must hold the private key for `from`; otherwise a forged
	// transfer (knowing only the public key) would be accepted.
	if err := verifyOwnerKey(from, privateKey); err != nil {
		return nil, err
	}

	timestamp := time.Now().Unix()
	message := fmt.Sprintf("%s|%s|%s|%d", nftID, base64.StdEncoding.EncodeToString(from), base64.StdEncoding.EncodeToString(to), timestamp)
	messageHash := sha256.Sum256([]byte(message))
	signature := ed25519.Sign(privateKey, messageHash[:])

	op := NewOperation(nftID, "transfer", from, to, signature)
	height, err := chain.AddBlock(fmt.Sprintf("%s|%s", op.ID, op.Type))
	if err != nil {
		return nil, err
	}
	op.BlockHeight = height
	op.Timestamp = timestamp

	// Persist the operation record and the ownership transfer as ONE
	// transaction. The conditional UPDATE inside TryTransferOwnership
	// remains the single source of truth for ownership: it rejects with
	// ErrOwnershipChanged if `from` no longer holds the NFT (concurrent
	// transfer won first). Pre-transaction, a rejected transfer left the
	// operation row behind as an orphan "attempt" record; with a real
	// transaction the whole unit rolls back, so nft_operations only ever
	// contains operations that actually applied.
	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		txRepo := s.txRepo(tx)
		if err := txRepo.SaveOperation(op); err != nil {
			return fmt.Errorf("failed to save transfer operation: %w", err)
		}
		return txRepo.TryTransferOwnership(nftID, from, to)
	})
	if err != nil {
		if errors.Is(err, ErrOwnershipChanged) {
			return nil, ErrNotOwner
		}
		return nil, err
	}
	return op, nil
}

func (s *NFTService) Burn(nftID string, owner, privateKey []byte, chain blockchain.BlockWriter) error {
	// Existence check only — same pattern as Transfer.
	nft, err := s.repo.GetNFT(nftID)
	if err != nil {
		return err
	}
	if nft == nil {
		return ErrNFTNotFound
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return ErrInvalidPrivateKey
	}
	if len(owner) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}
	// The caller must hold the private key for `owner`; otherwise a forged
	// burn (knowing only the public key) would be accepted.
	if err := verifyOwnerKey(owner, privateKey); err != nil {
		return err
	}

	timestamp := time.Now().Unix()
	message := fmt.Sprintf("%s|burn|%d", nftID, timestamp)
	messageHash := sha256.Sum256([]byte(message))
	signature := ed25519.Sign(privateKey, messageHash[:])

	op := NewOperation(nftID, "burn", owner, nil, signature)
	height, err := chain.AddBlock(fmt.Sprintf("%s|%s", op.ID, op.Type))
	if err != nil {
		return err
	}
	op.BlockHeight = height
	op.Timestamp = timestamp

	// Persist the burn operation and the conditional delete as ONE
	// transaction. TryDeleteNFTIfOwned remains the atomic ownership check:
	// only the caller that still holds the NFT succeeds, and a rejected
	// burn rolls back its operation record along with the failed delete.
	err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
		txRepo := s.txRepo(tx)
		if err := txRepo.SaveOperation(op); err != nil {
			return fmt.Errorf("failed to save burn operation: %w", err)
		}
		return txRepo.TryDeleteNFTIfOwned(nftID, owner)
	})
	if err != nil {
		if errors.Is(err, ErrOwnershipChanged) {
			return ErrNotOwner
		}
		return err
	}

	return nil
}

func (s *NFTService) VerifyTransfer(op *Operation) (bool, error) {
	if !op.IsTransfer() {
		return false, nil
	}

	message := fmt.Sprintf("%s|%s|%s|%d", op.NFTID, base64.StdEncoding.EncodeToString(op.From), base64.StdEncoding.EncodeToString(op.To), op.Timestamp)
	messageHash := sha256.Sum256([]byte(message))

	return ed25519.Verify(op.From, messageHash[:], op.Signature), nil
}

func (s *NFTService) GetNFTByID(id string) (*NFT, error) {
	return s.repo.GetNFT(id)
}

func (s *NFTService) GetNFTsByOwner(ownerPub []byte, limit, offset int) ([]*NFT, error) {
	return s.repo.GetNFTsByOwner(ownerPub, limit, offset)
}

func (s *NFTService) GetNFTsByCreator(creatorPub []byte) ([]*NFT, error) {
	return s.repo.GetNFTsByCreator(creatorPub)
}

func (s *NFTService) GetOperations(nftID string) ([]*Operation, error) {
	return s.repo.GetOperations(nftID)
}
