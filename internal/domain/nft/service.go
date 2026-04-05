package nft

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
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

func (s *NFTService) Mint(nft *NFT, chain blockchain.BlockWriter) (*NFT, error) {
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

	op := NewOperation(nft.ID, "mint", nil, nft.Owner, nil)
	op.BlockHeight = height
	s.repo.SaveOperation(op)

	return nft, nil
}

func (s *NFTService) Transfer(nftID string, from, to, privateKey []byte, chain blockchain.BlockWriter) (*Operation, error) {
	nft, err := s.repo.GetNFT(nftID)
	if err != nil {
		return nil, err
	}
	if nft == nil {
		return nil, ErrNFTNotFound
	}

	if !nft.IsOwner(from) {
		return nil, ErrNotOwner
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

	nft.Owner = to
	if err := s.repo.UpdateNFT(nft); err != nil {
		return nil, err
	}

	s.repo.SaveOperation(op)
	return op, nil
}

func (s *NFTService) Burn(nftID string, owner, privateKey []byte, chain blockchain.BlockWriter) error {
	nft, err := s.repo.GetNFT(nftID)
	if err != nil {
		return err
	}
	if nft == nil {
		return ErrNFTNotFound
	}

	if !nft.IsOwner(owner) {
		return ErrNotOwner
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

	if err := s.repo.DeleteNFT(nftID); err != nil {
		return err
	}

	s.repo.SaveOperation(op)
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
