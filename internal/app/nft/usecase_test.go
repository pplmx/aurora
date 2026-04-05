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

type mockBlockWriter struct {
	height int64
}

func (m *mockBlockWriter) AddBlock(data string) (int64, error) {
	m.height++
	return m.height, nil
}

func TestMintNFTUseCase_Execute(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}

	uc := NewMintNFTUseCase(service, chain)

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
	chain := &mockBlockWriter{}
	uc := NewMintNFTUseCase(service, chain)

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

func TestTransferNFTUseCase_Execute(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}
	uc := NewTransferNFTUseCase(service, chain)

	req := &TransferNFTRequest{
		NFTID:      "nft-123",
		From:       "ZnJvbS1waw==",
		To:         "dG8tcGs=",
		PrivateKey: "cHJpdmF0ZS1rZXk=",
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestTransferNFTUseCase_InvalidFrom(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}
	uc := NewTransferNFTUseCase(service, chain)

	req := &TransferNFTRequest{
		NFTID:      "nft-123",
		From:       "!!!invalid!!!",
		To:         "dG8tcGs=",
		PrivateKey: "cHJpdmF0ZS1rZXk=",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid from")
	}
}

func TestTransferNFTUseCase_InvalidTo(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}
	uc := NewTransferNFTUseCase(service, chain)

	req := &TransferNFTRequest{
		NFTID:      "nft-123",
		From:       "ZnJvbS1waw==",
		To:         "!!!invalid!!!",
		PrivateKey: "cHJpdmF0ZS1rZXk=",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid to")
	}
}

func TestTransferNFTUseCase_InvalidPrivateKey(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}
	uc := NewTransferNFTUseCase(service, chain)

	req := &TransferNFTRequest{
		NFTID:      "nft-123",
		From:       "ZnJvbS1waw==",
		To:         "dG8tcGs=",
		PrivateKey: "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid private key")
	}
}

func TestBurnNFTUseCase_Execute(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}
	uc := NewBurnNFTUseCase(service, chain)

	req := &BurnNFTRequest{
		NFTID:      "nft-123",
		Owner:      "b3duZXItcGs=",
		PrivateKey: "cHJpdmF0ZS1rZXk=",
	}

	err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestBurnNFTUseCase_InvalidOwner(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}
	uc := NewBurnNFTUseCase(service, chain)

	req := &BurnNFTRequest{
		NFTID:      "nft-123",
		Owner:      "!!!invalid!!!",
		PrivateKey: "cHJpdmF0ZS1rZXk=",
	}

	err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestBurnNFTUseCase_InvalidPrivateKey(t *testing.T) {
	service := &mockNFTService{}
	chain := &mockBlockWriter{}
	uc := NewBurnNFTUseCase(service, chain)

	req := &BurnNFTRequest{
		NFTID:      "nft-123",
		Owner:      "b3duZXItcGs=",
		PrivateKey: "!!!invalid!!!",
	}

	err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid private key")
	}
}

func TestGetNFTUseCase_Execute(t *testing.T) {
	service := &mockNFTService{
		nfts: []*nft.NFT{
			{ID: "nft-1", Name: "Test NFT"},
		},
	}
	uc := NewGetNFTUseCase(service)

	resp, err := uc.Execute("nft-1")
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

func TestGetNFTUseCase_NotFound(t *testing.T) {
	service := &mockNFTService{}
	uc := NewGetNFTUseCase(service)

	_, err := uc.Execute("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent NFT")
	}
}

func TestListNFTsByOwnerUseCase_Execute(t *testing.T) {
	service := &mockNFTService{
		nfts: []*nft.NFT{
			{ID: "nft-1", Name: "NFT 1"},
			{ID: "nft-2", Name: "NFT 2"},
		},
	}
	uc := NewListNFTsByOwnerUseCase(service)

	resp, err := uc.Execute("b3duZXItcGs=")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp) != 2 {
		t.Errorf("Expected 2 NFTs, got %d", len(resp))
	}
}

func TestListNFTsByOwnerUseCase_InvalidOwner(t *testing.T) {
	service := &mockNFTService{}
	uc := NewListNFTsByOwnerUseCase(service)

	_, err := uc.Execute("!!!invalid!!!")
	if err == nil {
		t.Fatal("Expected error for invalid owner")
	}
}

func TestGetNFTOperationsUseCase_Execute(t *testing.T) {
	service := &mockNFTService{}
	service.nfts = []*nft.NFT{{ID: "nft-1"}}

	uc := NewGetNFTOperationsUseCase(service)

	resp, err := uc.Execute("nft-1")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp) != 0 {
		t.Logf("Got %d operations (mock returns nil)", len(resp))
	}
}
