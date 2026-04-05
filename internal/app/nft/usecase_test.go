package nft

import (
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
)

type mockNFTService struct {
	nfts []*nft.NFT
}

func (m *mockNFTService) Mint(n *nft.NFT, chain blockchain.BlockWriter) (*nft.NFT, error) {
	n.ID = "test-id"
	n.Timestamp = time.Now().Unix()
	m.nfts = append(m.nfts, n)
	return n, nil
}

func (m *mockNFTService) Transfer(nftID string, from, to, privateKey []byte, chain blockchain.BlockWriter) (*nft.Operation, error) {
	return &nft.Operation{ID: "op-id"}, nil
}

func (m *mockNFTService) Burn(nftID string, owner, privateKey []byte, chain blockchain.BlockWriter) error {
	return nil
}

func (m *mockNFTService) VerifyTransfer(op *nft.Operation) (bool, error) {
	return true, nil
}

func (m *mockNFTService) GetNFTByID(id string) (*nft.NFT, error) {
	for _, n := range m.nfts {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, nil
}

func (m *mockNFTService) GetNFTsByOwner(ownerPub []byte) ([]*nft.NFT, error) {
	return m.nfts, nil
}

func (m *mockNFTService) GetNFTsByCreator(creatorPub []byte) ([]*nft.NFT, error) {
	return m.nfts, nil
}

func (m *mockNFTService) GetOperations(nftID string) ([]*nft.Operation, error) {
	return nil, nil
}

func TestMintNFTUseCase_Execute(t *testing.T) {
	service := &mockNFTService{}

	uc := NewMintNFTUseCase(service)

	req := &MintNFTRequest{
		Name:        "Test NFT",
		Description: "A test NFT",
		ImageURL:    "https://example.com/image.png",
		Creator:     "Y3JlYXRvci1waw==", // base64 of "creator-pk"
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Name != "Test NFT" {
		t.Errorf("Expected name 'Test NFT', got '%s'", resp.Name)
	}
}

func TestMintNFTUseCase_InvalidInput(t *testing.T) {
	service := &mockNFTService{}
	uc := NewMintNFTUseCase(service)

	tests := []struct {
		name    string
		req     *MintNFTRequest
		wantErr bool
	}{
		{
			name: "empty name",
			req: &MintNFTRequest{
				Name:    "",
				Creator: "Y3JlYXRvci1waw==",
			},
			wantErr: true,
		},
		{
			name: "invalid creator base64",
			req: &MintNFTRequest{
				Name:    "Test NFT",
				Creator: "!!!invalid!!!",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Execute(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
