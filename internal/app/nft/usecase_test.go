package nft

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/stretchr/testify/require"
)

type mockNFTService struct {
	nfts []*nft.NFT
}

// Real-length key fixtures: the app layer now rejects wrong-length keys
// (TASK-112), so callers must present 32-byte public / 64-byte private keys.
var (
	testPubB64     = "QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUE="
	testPrivB64    = "QkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQg=="
	testCreatorB64 = "Q0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0M="
)

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

func (m *mockNFTService) GetNFTsByOwner(ownerPub []byte, limit, offset int) ([]*nft.NFT, error) {
	all := m.nfts
	if limit > 0 {
		if offset < 0 {
			offset = 0
		}
		if offset >= len(all) {
			return nil, nil
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		all = all[offset:end]
	}
	return all, nil
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
		Creator:     testCreatorB64,
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

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
				Creator: testCreatorB64,
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
		From:       testPubB64,
		To:         testPubB64,
		PrivateKey: testPrivB64,
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
		To:         testPubB64,
		PrivateKey: testPrivB64,
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
		From:       testPubB64,
		To:         "!!!invalid!!!",
		PrivateKey: testPrivB64,
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
		From:       testPubB64,
		To:         testPubB64,
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
		Owner:      testPubB64,
		PrivateKey: testPrivB64,
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
		PrivateKey: testPrivB64,
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
		Owner:      testPubB64,
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
	require.NoError(t, err)
	require.NotNil(t, resp)

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

	resp, err := uc.Execute(&ListNFTsByOwnerRequest{Owner: testPubB64})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp) != 2 {
		t.Errorf("Expected 2 NFTs, got %d", len(resp))
	}
}

func TestListNFTsByOwnerUseCase_Paged(t *testing.T) {
	service := &mockNFTService{nfts: []*nft.NFT{}}
	for i := 0; i < 5; i++ {
		service.nfts = append(service.nfts, &nft.NFT{ID: fmt.Sprintf("nft-%d", i)})
	}
	uc := NewListNFTsByOwnerUseCase(service)

	// limit/offset must be honored at the use-case layer (TASK-101, ISS-093).
	page, err := uc.Execute(&ListNFTsByOwnerRequest{Owner: testPubB64, Limit: 2, Offset: 1})
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "nft-1", page[0].ID)
	require.Equal(t, "nft-2", page[1].ID)

	// Offset past the end is an empty page, not an error.
	page, err = uc.Execute(&ListNFTsByOwnerRequest{Owner: testPubB64, Limit: 10, Offset: 100})
	require.NoError(t, err)
	require.Len(t, page, 0)
}

func TestListNFTsByOwnerUseCase_InvalidOwner(t *testing.T) {
	service := &mockNFTService{}
	uc := NewListNFTsByOwnerUseCase(service)

	_, err := uc.Execute(&ListNFTsByOwnerRequest{Owner: "!!!invalid!!!"})
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

// TestMintNFTUseCase_RejectsShortCreatorKey is the regression test for the
// bricked-NFT hole (TASK-112, ISS-104): mint used to accept any number of
// decoded key bytes, so `nft mint -c <short-base64>` stored a wrong-length
// owner and produced an NFT that could never be transferred or burned (the
// domain's atomic ownership checks need a 32-byte key and can never match),
// reported as "minted successfully". A wrong-length creator must now be a
// client error.
func TestMintNFTUseCase_RejectsShortCreatorKey(t *testing.T) {
	uc := NewMintNFTUseCase(&mockNFTService{}, &mockBlockWriter{})

	shortKey := base64.StdEncoding.EncodeToString([]byte("short-key"))
	_, err := uc.Execute(&MintNFTRequest{Name: "X", Creator: shortKey})
	require.ErrorIs(t, err, nft.ErrInvalidPublicKey)

	// A zero-length (empty) key is also rejected, not minted.
	_, err = uc.Execute(&MintNFTRequest{Name: "X", Creator: ""})
	require.Error(t, err)
}

// TestToNFTResponse_KeysAreBase64 is the regression test for the unreadable
// key output (TASK-112, ISS-104): the NFT response previously emitted raw
// `string(nft.Owner)` bytes (mojibake in the CLI and JSON), unlike every other
// module which base64-encodes keys.
func TestToNFTResponse_KeysAreBase64(t *testing.T) {
	raw := bytes.Repeat([]byte{0xFF}, 32)
	resp := ToNFTResponse(&nft.NFT{ID: "n1", Name: "N", Owner: raw, Creator: raw})
	require.NotNil(t, resp)
	require.Equal(t, base64.StdEncoding.EncodeToString(raw), resp.Owner)
	require.Equal(t, base64.StdEncoding.EncodeToString(raw), resp.Creator)
	// the raw bytes must never appear as-is
	require.NotEqual(t, string(raw), resp.Owner)
}
