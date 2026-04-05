package nft

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pplmx/aurora/internal/blockchain"
)

type NFT struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	Creator     string `json:"creator"`
	Owner       string `json:"owner"`
	TokenURI    string `json:"token_uri"`
	BlockHeight int64  `json:"block_height"`
	Timestamp   int64  `json:"timestamp"`
}

type NFTOperation struct {
	ID          string `json:"id"`
	NFTID       string `json:"nft_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	Operation   string `json:"operation"`
	Signature   string `json:"signature"`
	BlockHeight int64  `json:"block_height"`
	Timestamp   int64  `json:"timestamp"`
}

type NFTStorage struct {
	nfts         map[string]*NFT
	operations   map[string][]*NFTOperation
	ownerIndex   map[string][]string
	creatorIndex map[string][]string
}

func NewNFTStorage() *NFTStorage {
	return &NFTStorage{
		nfts:         make(map[string]*NFT),
		operations:   make(map[string][]*NFTOperation),
		ownerIndex:   make(map[string][]string),
		creatorIndex: make(map[string][]string),
	}
}

func (s *NFTStorage) SaveNFT(nft *NFT) error {
	s.nfts[nft.ID] = nft
	s.ownerIndex[nft.Owner] = append(s.ownerIndex[nft.Owner], nft.ID)
	s.creatorIndex[nft.Creator] = append(s.creatorIndex[nft.Creator], nft.ID)
	return nil
}

func (s *NFTStorage) GetNFT(id string) (*NFT, error) {
	return s.nfts[id], nil
}

func (s *NFTStorage) UpdateNFT(nft *NFT) error {
	stored, exists := s.nfts[nft.ID]
	if !exists {
		return fmt.Errorf("NFT not found")
	}

	storedCopy := *stored
	oldOwner := storedCopy.Owner
	newOwner := nft.Owner
	if oldOwner != newOwner {
		oldList := s.ownerIndex[oldOwner]
		for i, nid := range oldList {
			if nid == nft.ID {
				s.ownerIndex[oldOwner] = append(oldList[:i], oldList[i+1:]...)
				break
			}
		}
		s.ownerIndex[newOwner] = append(s.ownerIndex[newOwner], nft.ID)
	}

	s.nfts[nft.ID] = nft
	return nil
}

func (s *NFTStorage) DeleteNFT(id string) error {
	nft, exists := s.nfts[id]
	if !exists {
		return nil
	}

	oldList := s.ownerIndex[nft.Owner]
	for i, nid := range oldList {
		if nid == id {
			s.ownerIndex[nft.Owner] = append(oldList[:i], oldList[i+1:]...)
			break
		}
	}

	delete(s.nfts, id)
	return nil
}

func (s *NFTStorage) GetNFTsByOwner(owner string) ([]*NFT, error) {
	ids := s.ownerIndex[owner]
	result := make([]*NFT, 0, len(ids))
	for _, id := range ids {
		if nft, exists := s.nfts[id]; exists {
			result = append(result, nft)
		}
	}
	return result, nil
}

func (s *NFTStorage) GetNFTsByCreator(creator string) ([]*NFT, error) {
	ids := s.creatorIndex[creator]
	result := make([]*NFT, 0, len(ids))
	for _, id := range ids {
		if nft, exists := s.nfts[id]; exists {
			result = append(result, nft)
		}
	}
	return result, nil
}

func (s *NFTStorage) SaveOperation(op *NFTOperation) error {
	s.operations[op.NFTID] = append(s.operations[op.NFTID], op)
	return nil
}

func (s *NFTStorage) GetOperations(nftID string) ([]*NFTOperation, error) {
	return s.operations[nftID], nil
}

var nftStorage *NFTStorage

func SetNFTStorage(s *NFTStorage) {
	nftStorage = s
}

func MintNFT(name, description, imageURL, tokenURI string, creatorPub []byte, chain *blockchain.BlockChain) (*NFT, error) {
	if nftStorage == nil {
		return nil, fmt.Errorf("NFT storage not initialized")
	}

	nft := &NFT{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		ImageURL:    imageURL,
		Creator:     base64.StdEncoding.EncodeToString(creatorPub),
		Owner:       base64.StdEncoding.EncodeToString(creatorPub),
		TokenURI:    tokenURI,
		Timestamp:   time.Now().Unix(),
	}

	jsonData, _ := json.Marshal(nft)
	height := chain.AddBlock(string(jsonData))
	nft.BlockHeight = height

	if err := nftStorage.SaveNFT(nft); err != nil {
		return nil, err
	}

	op := &NFTOperation{
		ID:          uuid.New().String(),
		NFTID:       nft.ID,
		From:        "",
		To:          nft.Owner,
		Operation:   "mint",
		BlockHeight: height,
		Timestamp:   nft.Timestamp,
	}
	nftStorage.SaveOperation(op)

	return nft, nil
}

func TransferNFT(nftID string, fromPub, fromPriv, toPub []byte, chain *blockchain.BlockChain) (*NFTOperation, error) {
	if nftStorage == nil {
		return nil, fmt.Errorf("NFT storage not initialized")
	}

	nft, err := nftStorage.GetNFT(nftID)
	if err != nil {
		return nil, err
	}
	if nft == nil {
		return nil, fmt.Errorf("NFT not found")
	}

	fromPubStr := base64.StdEncoding.EncodeToString(fromPub)
	if nft.Owner != fromPubStr {
		return nil, fmt.Errorf("not the owner")
	}

	timestamp := time.Now().Unix()
	message := fmt.Sprintf("%s|%s|%s|%d", nftID, fromPubStr, base64.StdEncoding.EncodeToString(toPub), timestamp)
	messageHash := sha256.Sum256([]byte(message))
	signature := ed25519.Sign(fromPriv, messageHash[:])

	toPubStr := base64.StdEncoding.EncodeToString(toPub)
	op := &NFTOperation{
		ID:        uuid.New().String(),
		NFTID:     nftID,
		From:      fromPubStr,
		To:        toPubStr,
		Operation: "transfer",
		Signature: base64.StdEncoding.EncodeToString(signature),
		Timestamp: timestamp,
	}

	jsonData, _ := json.Marshal(op)
	height := chain.AddBlock(string(jsonData))
	op.BlockHeight = height

	oldOwner := nft.Owner
	nft.Owner = toPubStr

	oldList := nftStorage.ownerIndex[oldOwner]
	for i, nid := range oldList {
		if nid == nftID {
			nftStorage.ownerIndex[oldOwner] = append(oldList[:i], oldList[i+1:]...)
			break
		}
	}
	nftStorage.ownerIndex[toPubStr] = append(nftStorage.ownerIndex[toPubStr], nftID)

	nftStorage.SaveOperation(op)

	return op, nil
}

func VerifyTransfer(op *NFTOperation) bool {
	pubBytes, _ := base64.StdEncoding.DecodeString(op.From)
	sigBytes, _ := base64.StdEncoding.DecodeString(op.Signature)
	message := fmt.Sprintf("%s|%s|%s|%d", op.NFTID, op.From, op.To, op.Timestamp)
	messageHash := sha256.Sum256([]byte(message))
	return ed25519.Verify(pubBytes, messageHash[:], sigBytes)
}

func BurnNFT(nftID string, ownerPub, ownerPriv []byte, chain *blockchain.BlockChain) error {
	if nftStorage == nil {
		return fmt.Errorf("NFT storage not initialized")
	}

	nft, err := nftStorage.GetNFT(nftID)
	if err != nil {
		return err
	}
	if nft == nil {
		return fmt.Errorf("NFT not found")
	}

	ownerPubStr := base64.StdEncoding.EncodeToString(ownerPub)
	if nft.Owner != ownerPubStr {
		return fmt.Errorf("not the owner")
	}

	timestamp := time.Now().Unix()
	message := fmt.Sprintf("%s|burn|%d", nftID, timestamp)
	messageHash := sha256.Sum256([]byte(message))
	signature := ed25519.Sign(ownerPriv, messageHash[:])

	op := &NFTOperation{
		ID:        uuid.New().String(),
		NFTID:     nftID,
		From:      ownerPubStr,
		To:        "",
		Operation: "burn",
		Signature: base64.StdEncoding.EncodeToString(signature),
		Timestamp: timestamp,
	}

	jsonData, _ := json.Marshal(op)
	height := chain.AddBlock(string(jsonData))
	op.BlockHeight = height

	nftStorage.DeleteNFT(nftID)
	nftStorage.SaveOperation(op)

	return nil
}

func GetNFTByID(id string) (*NFT, error) {
	return nftStorage.GetNFT(id)
}

func GetNFTsByOwner(ownerPub []byte) ([]*NFT, error) {
	return nftStorage.GetNFTsByOwner(base64.StdEncoding.EncodeToString(ownerPub))
}

func GetNFTsByCreator(creatorPub []byte) ([]*NFT, error) {
	return nftStorage.GetNFTsByCreator(base64.StdEncoding.EncodeToString(creatorPub))
}

func GetNFTOperations(nftID string) ([]*NFTOperation, error) {
	return nftStorage.GetOperations(nftID)
}
