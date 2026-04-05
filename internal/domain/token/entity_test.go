package token

import (
	"testing"
)

func TestNewAmount(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected string
	}{
		{"zero", 0, "0"},
		{"positive", 100, "100"},
		{"negative", -50, "-50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := NewAmount(tt.value)
			if amount.String() != tt.expected {
				t.Errorf("NewAmount(%d).String() = %s, want %s", tt.value, amount.String(), tt.expected)
			}
		})
	}
}

func TestNewAmountFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		expectStr string
	}{
		{"valid positive", "1000", false, "1000"},
		{"valid zero", "0", false, "0"},
		{"valid negative", "-500", false, "-500"},
		{"valid large number", "999999999999999999999999999999", false, "999999999999999999999999999999"},
		{"invalid empty", "", true, ""},
		{"invalid non-number", "abc", true, ""},
		{"invalid float", "1.5", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := NewAmountFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAmountFromString(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && amount.String() != tt.expectStr {
				t.Errorf("NewAmountFromString(%s) = %s, want %s", tt.input, amount.String(), tt.expectStr)
			}
		})
	}
}

func TestAmount_IsPositive(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected bool
	}{
		{"positive", 100, true},
		{"zero", 0, false},
		{"negative", -50, false},
		{"nil amount", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var amount *Amount
			if tt.value != 0 || tt.name != "nil amount" {
				amount = NewAmount(tt.value)
			}
			if amount.IsPositive() != tt.expected {
				t.Errorf("Amount(%d).IsPositive() = %v, want %v", tt.value, amount.IsPositive(), tt.expected)
			}
		})
	}
}

func TestNewToken(t *testing.T) {
	owner := PublicKey("owner-public-key-12345678901234")
	supply := NewAmount(1000000)

	token := NewToken("token-123", "Test Token", "TEST", supply, owner)

	if token.ID() != "token-123" {
		t.Errorf("Token.ID() = %s, want token-123", token.ID())
	}
	if token.Name() != "Test Token" {
		t.Errorf("Token.Name() = %s, want Test Token", token.Name())
	}
	if token.Symbol() != "TEST" {
		t.Errorf("Token.Symbol() = %s, want TEST", token.Symbol())
	}
	if token.TotalSupply().String() != "1000000" {
		t.Errorf("Token.TotalSupply() = %s, want 1000000", token.TotalSupply().String())
	}
	if token.Decimals() != 8 {
		t.Errorf("Token.Decimals() = %d, want 8", token.Decimals())
	}
	if string(token.Owner()) != string(owner) {
		t.Errorf("Token.Owner() = %s, want %s", string(token.Owner()), string(owner))
	}
	if !token.IsMintable() {
		t.Error("Token.IsMintable() = false, want true")
	}
	if !token.IsBurnable() {
		t.Error("Token.IsBurnable() = false, want true")
	}
}

func TestNewApproval(t *testing.T) {
	owner := PublicKey("owner-pk-12345678901234567890")
	spender := PublicKey("spender-pk-1234567890123")
	amount := NewAmount(500)

	approval := NewApproval("token-123", owner, spender, amount)

	if approval.TokenID() != "token-123" {
		t.Errorf("Approval.TokenID() = %s, want token-123", approval.TokenID())
	}
	if string(approval.Owner()) != string(owner) {
		t.Errorf("Approval.Owner() = %s, want %s", string(approval.Owner()), string(owner))
	}
	if string(approval.Spender()) != string(spender) {
		t.Errorf("Approval.Spender() = %s, want %s", string(approval.Spender()), string(spender))
	}
	if approval.Amount().String() != "500" {
		t.Errorf("Approval.Amount() = %s, want 500", approval.Amount().String())
	}
}

func TestValidateTokenName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "My Token", false},
		{"empty name", "", true},
		{"name too long", string(make([]byte, 101)), true},
		{"max length name", string(make([]byte, 100)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTokenName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTokenName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTokenSymbol(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid symbol", "TKN", false},
		{"empty symbol", "", true},
		{"symbol too long", "TOOLONG1234", true},
		{"max length symbol", "ABCDEFGHIJ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTokenSymbol(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTokenSymbol(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  *Amount
		wantErr bool
	}{
		{"valid positive", NewAmount(100), false},
		{"zero amount", NewAmount(0), true},
		{"negative amount", NewAmount(-10), true},
		{"nil amount", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAmount(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAmount(%v) error = %v, wantErr %v", tt.amount, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePublicKey(t *testing.T) {
	tests := []struct {
		name    string
		pk      PublicKey
		wantErr bool
	}{
		{"valid 32-byte key", PublicKey(make([]byte, 32)), false},
		{"empty key", PublicKey{}, true},
		{"key too short", PublicKey("short"), true},
		{"key too long", PublicKey(make([]byte, 33)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublicKey(tt.pk)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePublicKey(%v) error = %v, wantErr %v", tt.pk, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNonce(t *testing.T) {
	tests := []struct {
		name         string
		nonce        uint64
		currentNonce uint64
		wantErr      bool
	}{
		{"valid nonce", 10, 5, false},
		{"nonce equal to current", 5, 5, true},
		{"nonce less than current", 3, 5, true},
		{"nonce is zero", 0, 0, true},
		{"nonce is one higher", 6, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonce(tt.nonce, tt.currentNonce)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNonce(%d, %d) error = %v, wantErr %v", tt.nonce, tt.currentNonce, err, tt.wantErr)
			}
		})
	}
}
