package nft

import (
	"testing"
)

func TestNFT_Validate(t *testing.T) {
	tests := []struct {
		name    string
		nft     *NFT
		wantErr bool
	}{
		{
			name: "valid NFT",
			nft: &NFT{
				Name:  "Test NFT",
				Owner: []byte("owner-pk"),
			},
			wantErr: false,
		},
		{
			name: "empty name",
			nft: &NFT{
				Name:  "",
				Owner: []byte("owner-pk"),
			},
			wantErr: true,
		},
		{
			name: "empty owner",
			nft: &NFT{
				Name:  "Test NFT",
				Owner: []byte{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.nft.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNFT_IsOwner(t *testing.T) {
	nft := &NFT{
		Owner: []byte("owner-pk"),
	}

	tests := []struct {
		name     string
		pubKey   []byte
		expected bool
	}{
		{"is owner", []byte("owner-pk"), true},
		{"not owner", []byte("other-pk"), false},
		{"empty pubKey", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nft.IsOwner(tt.pubKey)
			if result != tt.expected {
				t.Errorf("IsOwner() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOperation_IsTransfer(t *testing.T) {
	transferOp := &Operation{Type: "transfer"}
	mintOp := &Operation{Type: "mint"}

	if !transferOp.IsTransfer() {
		t.Error("transfer operation should return true for IsTransfer()")
	}
	if mintOp.IsTransfer() {
		t.Error("mint operation should return false for IsTransfer()")
	}
}

func TestOperation_IsBurn(t *testing.T) {
	burnOp := &Operation{Type: "burn"}
	transferOp := &Operation{Type: "transfer"}

	if !burnOp.IsBurn() {
		t.Error("burn operation should return true for IsBurn()")
	}
	if transferOp.IsBurn() {
		t.Error("transfer operation should return false for IsBurn()")
	}
}
