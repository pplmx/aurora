package nft

import (
	"crypto/ed25519"
	"crypto/sha256"
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
	GetNFTsByOwner(ownerPub []byte) ([]*NFT, error)
	GetNFTsByCreator(creatorPub []byte) ([]*NFT, error)
	GetOperations(nftID string) ([]*Operation, error)
}

type NFTService struct {
	repo Repository
}

func NewService(repo Repository) *NFTService {
	return &NFTService{repo: repo}
}

func (s *NFTService) Mint(nft *NFT, chain blockchain.BlockWriter) (_ *NFT, err error) {
	nft.ID = uuid.New().String()
	nft.Timestamp = time.Now().Unix()

	data := fmt.Sprintf("%s|%s|%s", nft.ID, nft.Name, nft.Owner)
	height, err := chain.AddBlock(data)
	if err != nil {
		return nil, err
	}
	nft.BlockHeight = height

	if err := s.repo.SaveNFT(nft); err != nil {
		return nil, err
	}
	// If we return an error after this point, the chain block is
	// already committed (cannot be reverted), but we must not
	// leave the NFT row orphaned without its audit operation.
	// Deleting the NFT leaves the chain block as an empty
	// tombstone, which is safe (no ownership claimable).
	defer func() {
		if err != nil {
			_ = s.repo.DeleteNFT(nft.ID)
		}
	}()

	op := NewOperation(nft.ID, "mint", nil, nft.Owner, nil)
	op.BlockHeight = height
	if err := s.repo.SaveOperation(op); err != nil {
		return nil, fmt.Errorf("failed to save mint operation: %w", err)
	}

	return nft, nil
}

func (s *NFTService) Transfer(nftID string, from, to, privateKey []byte, chain blockchain.BlockWriter) (_ *Operation, err error) {
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

	// Persist the operation BEFORE transferring ownership so a
	// failed transfer does not produce an orphan audit record.
	// If this save fails, nothing has been mutated yet — clean
	// abort.
	if err := s.repo.SaveOperation(op); err != nil {
		return nil, fmt.Errorf("failed to save transfer operation: %w", err)
	}

	// Atomic ownership transfer. The conditional UPDATE inside
	// the primitive rejects us if `from` no longer holds the
	// NFT (e.g. a concurrent transfer has already moved it).
	// If TryTransferOwnership fails after SaveOperation
	// succeeded, the operation record remains as an orphan —
	// this is safe: it records a transfer attempt that was
	// rejected due to concurrent ownership change.
	if err := s.repo.TryTransferOwnership(nftID, from, to); err != nil {
		if errors.Is(err, ErrOwnershipChanged) {
			return nil, ErrNotOwner
		}
		return nil, err
	}
	return op, nil
}

func (s *NFTService) Burn(nftID string, owner, privateKey []byte, chain blockchain.BlockWriter) (err error) {
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

	// Persist the operation BEFORE deleting the NFT so a
	// failed delete does not produce a deleted NFT with no
	// audit record. If this save fails, nothing has been
	// mutated yet — clean abort.
	if err := s.repo.SaveOperation(op); err != nil {
		return fmt.Errorf("failed to save burn operation: %w", err)
	}

	// Atomic delete: only the caller that still holds the NFT
	// succeeds. If TryDeleteNFTIfOwned fails after the operation
	// was saved, the operation record remains as an orphan, which
	// is safe — it records a burn attempt that was rejected due
	// to concurrent ownership change.
	if err := s.repo.TryDeleteNFTIfOwned(nftID, owner); err != nil {
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

func (s *NFTService) GetNFTsByOwner(ownerPub []byte) ([]*NFT, error) {
	return s.repo.GetNFTsByOwner(ownerPub)
}

func (s *NFTService) GetNFTsByCreator(creatorPub []byte) ([]*NFT, error) {
	return s.repo.GetNFTsByCreator(creatorPub)
}

func (s *NFTService) GetOperations(nftID string) ([]*Operation, error) {
	return s.repo.GetOperations(nftID)
}
