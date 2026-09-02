package nft

import (
	"strings"
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
		// Length-bound cases (TASK-271, ISS-267): mint free-text fields are
		// bounded like the token surface so a key-holding caller cannot grow
		// rows/responses without limit.
		{
			name: "name too long",
			nft: &NFT{
				Name:  strings.Repeat("n", MaxNFTNameLength+1),
				Owner: []byte("owner-pk"),
			},
			wantErr: true,
		},
		{
			name: "description too long",
			nft: &NFT{
				Name:        "Test NFT",
				Description: strings.Repeat("d", MaxNFTDescriptionLength+1),
				Owner:       []byte("owner-pk"),
			},
			wantErr: true,
		},
		{
			name: "image url too long",
			nft: &NFT{
				Name:      "Test NFT",
				ImageURL:  strings.Repeat("u", MaxNFTImageURLLength+1),
				Owner:     []byte("owner-pk"),
			},
			wantErr: true,
		},
		{
			name: "token uri too long",
			nft: &NFT{
				Name:      "Test NFT",
				TokenURI:  strings.Repeat("t", MaxNFTTokenURILength+1),
				Owner:     []byte("owner-pk"),
			},
			wantErr: true,
		},
		{
			name: "all fields at exact bounds",
			nft: &NFT{
				Name:        strings.Repeat("n", MaxNFTNameLength),
				Description: strings.Repeat("d", MaxNFTDescriptionLength),
				ImageURL:    strings.Repeat("u", MaxNFTImageURLLength),
				TokenURI:    strings.Repeat("t", MaxNFTTokenURILength),
				Owner:       []byte("owner-pk"),
			},
			wantErr: false,
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

// TestOperation_NewOperation_AssignsUniqueID locks ISS-072 / TASK-080: every
// operation created through the real path must carry its own non-empty UUID
// id, otherwise the SQLite PRIMARY-KEY INSERT OR REPLACE collapses the entire
// per-NFT audit history to a single ""-id row.
func TestOperation_NewOperation_AssignsUniqueID(t *testing.T) {
	a := NewOperation("nft-1", "transfer", []byte("from"), []byte("to"), nil)
	b := NewOperation("nft-1", "transfer", []byte("from"), []byte("to"), nil)

	if a.ID == "" || b.ID == "" {
		t.Fatalf("NewOperation must assign a non-empty id (got a=%q b=%q)", a.ID, b.ID)
	}
	if a.ID == b.ID {
		t.Fatalf("each operation must get its own id (got duplicated %q)", a.ID)
	}
}
