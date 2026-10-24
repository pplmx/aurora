package voting

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestEd25519Service_SignVote(t *testing.T) {
	service := NewEd25519Service()

	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	message := "test-vote-message"
	signature, err := service.SignVote(message, priv)
	if err != nil {
		t.Fatalf("SignVote failed: %v", err)
	}

	if signature == "" {
		t.Error("Expected non-empty signature")
	}
}

func TestEd25519Service_SignVote_InvalidKeySize(t *testing.T) {
	service := NewEd25519Service()

	_, err := service.SignVote("message", []byte("short-key"))
	if err == nil {
		t.Fatal("Expected error for invalid key size")
	}
}

func TestEd25519Service_VerifyVote_Empty(t *testing.T) {
	service := NewEd25519Service()

	valid := service.VerifyVote("", "message", "")
	if valid {
		t.Error("Expected invalid for empty strings")
	}
}

func TestEd25519Service_SignAndVerify(t *testing.T) {
	service := NewEd25519Service()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	message := "vote-for-alice"
	signature, err := service.SignVote(message, priv)
	if err != nil {
		t.Fatalf("SignVote failed: %v", err)
	}

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	valid := service.VerifyVote(pubB64, message, signature)
	if !valid {
		t.Error("Expected valid signature")
	}
}
