package token

import (
	"crypto/ed25519"
	"crypto/rand"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewTokenFromRecord(t *testing.T) {
	owner := PublicKey("owner-key")
	created := time.Unix(1000, 0)
	tok := NewTokenFromRecord("T1", "Name", "SYM", NewAmount(1000), owner, 8, true, false, created)
	if tok.ID() != "T1" || tok.Name() != "Name" || tok.Symbol() != "SYM" {
		t.Errorf("unexpected token metadata: %q %q %q", tok.ID(), tok.Name(), tok.Symbol())
	}
	if tok.TotalSupply().String() != "1000" {
		t.Errorf("totalSupply = %s, want 1000", tok.TotalSupply().String())
	}
	if tok.Decimals() != 8 || !tok.IsMintable() || tok.IsBurnable() {
		t.Errorf("unexpected flags: decimals=%d mintable=%v burnable=%v", tok.Decimals(), tok.IsMintable(), tok.IsBurnable())
	}
	if !tok.CreatedAt().Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", tok.CreatedAt(), created)
	}
}

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
		{"max int64 accepted", &Amount{big.NewInt(math.MaxInt64)}, false},
		{"zero amount", NewAmount(0), true},
		{"negative amount", NewAmount(-10), true},
		{"nil amount", nil, true},
		{"exceeds max int64 rejected", &Amount{new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1))}, true},
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

func TestAmount_String(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected string
	}{
		{"positive", 100, "100"},
		{"zero", 0, "0"},
		{"negative", -50, "-50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := NewAmount(tt.value)
			if amount.String() != tt.expected {
				t.Errorf("Amount(%d).String() = %s, want %s", tt.value, amount.String(), tt.expected)
			}
		})
	}

	t.Run("nil amount", func(t *testing.T) {
		var amount *Amount = nil
		if amount.String() != "0" {
			t.Errorf("nil Amount.String() = %s, want 0", amount.String())
		}
	})

	t.Run("nil int", func(t *testing.T) {
		amount := &Amount{Int: nil}
		if amount.String() != "0" {
			t.Errorf("Amount with nil Int.String() = %s, want 0", amount.String())
		}
	})
}

func TestAmount_Cmp(t *testing.T) {
	tests := []struct {
		name     string
		a        *Amount
		b        *Amount
		expected int
	}{
		{"equal positive", NewAmount(100), NewAmount(100), 0},
		{"a greater", NewAmount(100), NewAmount(50), 1},
		{"b greater", NewAmount(50), NewAmount(100), -1},
		{"both nil", nil, nil, 0},
		{"a nil", nil, NewAmount(100), -1},
		{"b nil", NewAmount(100), nil, 1},
		// A non-nil Amount whose embedded integer was never set reads as 0
		// everywhere else (String, Sign, Int64). Comparing one must give the
		// same answer instead of dereferencing the nil integer.
		{"zero value vs zero value", &Amount{}, &Amount{}, 0},
		{"zero value vs positive", &Amount{}, NewAmount(100), -1},
		{"positive vs zero value", NewAmount(100), &Amount{}, 1},
		{"nil receiver vs zero value", nil, &Amount{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Cmp(tt.b)
			if result != tt.expected {
				t.Errorf("Cmp(%v, %v) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestAmount_Sign(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected int
	}{
		{"positive", 100, 1},
		{"zero", 0, 0},
		{"negative", -50, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := NewAmount(tt.value)
			if amount.Sign() != tt.expected {
				t.Errorf("Amount(%d).Sign() = %d, want %d", tt.value, amount.Sign(), tt.expected)
			}
		})
	}

	t.Run("nil amount", func(t *testing.T) {
		var amount *Amount = nil
		if amount.Sign() != 0 {
			t.Errorf("nil Amount.Sign() = %d, want 0", amount.Sign())
		}
	})

	t.Run("nil int", func(t *testing.T) {
		amount := &Amount{Int: nil}
		if amount.Sign() != 0 {
			t.Errorf("Amount with nil Int.Sign() = %d, want 0", amount.Sign())
		}
	})
}

func TestAmount_Int64(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected int64
	}{
		{"positive", 100, 100},
		{"zero", 0, 0},
		{"negative", -50, -50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := NewAmount(tt.value)
			if amount.Int64() != tt.expected {
				t.Errorf("Amount(%d).Int64() = %d, want %d", tt.value, amount.Int64(), tt.expected)
			}
		})
	}

	t.Run("nil amount", func(t *testing.T) {
		var amount *Amount = nil
		if amount.Int64() != 0 {
			t.Errorf("nil Amount.Int64() = %d, want 0", amount.Int64())
		}
	})

	t.Run("nil int", func(t *testing.T) {
		amount := &Amount{Int: nil}
		if amount.Int64() != 0 {
			t.Errorf("Amount with nil Int.Int64() = %d, want 0", amount.Int64())
		}
	})
}

func TestToken_CreatedAt(t *testing.T) {
	owner := PublicKey("owner-public-key-12345678901234")
	supply := NewAmount(1000000)

	token := NewToken("token-123", "Test Token", "TEST", supply, owner)

	if token.CreatedAt().IsZero() {
		t.Error("Token.CreatedAt() should not be zero")
	}
}

func TestApproval_ExpiresAt(t *testing.T) {
	owner := PublicKey("owner-pk-12345678901234567890")
	spender := PublicKey("spender-pk-1234567890123")
	amount := NewAmount(500)

	approval := NewApproval("token-123", owner, spender, amount)

	if !approval.ExpiresAt().IsZero() {
		t.Errorf("Approval.ExpiresAt() = %v, want zero time", approval.ExpiresAt())
	}
}

// TestNewAmountFromStringSaysWhatItExpects pins the message a reader sees.
//
// The parse error is the only feedback a form or CLI flag gets when the value
// cannot be stored, and the amount is a whole number of base units: a decimal or
// a grouped "1,000" is refused rather than rounded. "invalid amount" alone left
// the reader unable to tell which part of the value was wrong.
func TestNewAmountFromStringSaysWhatItExpects(t *testing.T) {
	for _, input := range []string{"1.5", "1,000", "abc"} {
		_, err := NewAmountFromString(input)
		require.ErrorIs(t, err, ErrInvalidAmount, "callers still match the sentinel")
		require.Contains(t, err.Error(), input,
			"the message quotes what was typed, so the reader can see which field is wrong")
		require.Contains(t, err.Error(), "whole number",
			"and says what is expected instead")
	}

	_, err := NewAmountFromString("")
	require.ErrorIs(t, err, ErrInvalidAmount)
	require.Contains(t, err.Error(), "whole number of base units",
		"an empty value gets the rule, not a quote of nothing")
}

// TestTheUnauthorizedMessageNamesTheProblem pins the sentence a reader gets when
// the presented private key is not the one the operation needs. "unauthorized"
// said the key was rejected but not why, and the wrong-key mistake is the one a
// reader can fix — the NFT module already words it this way.
func TestTheUnauthorizedMessageNamesTheProblem(t *testing.T) {
	require.Contains(t, ErrUnauthorized.Error(), "private key",
		"the message should say the key is the problem")
	require.Contains(t, ErrUnauthorized.Error(), "does not match",
		"and that it is a mismatch rather than a broader refusal")
	require.NotEqual(t, "unauthorized", ErrUnauthorized.Error(),
		"the bare word left the reader to guess")

	// The sentinel is what callers match on, so it must keep working. Two freshly
	// generated pairs never share a public key.
	_, ownerPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	otherPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	require.ErrorIs(t, VerifyPrivateKeyMatches(PublicKey(otherPub), PrivateKey(ownerPriv)),
		ErrUnauthorized, "callers still match the sentinel, whatever the message says")
}
