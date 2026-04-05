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

func TestEd25519Service_CountVotes(t *testing.T) {
	service := NewEd25519Service()

	candidates := []Candidate{
		{ID: "c1", Name: "Alice", VoteCount: 10},
		{ID: "c2", Name: "Bob", VoteCount: 5},
		{ID: "c3", Name: "Charlie", VoteCount: 15},
	}

	results := service.CountVotes(candidates)

	if results["c1"] != 10 {
		t.Errorf("Expected 10 votes for c1, got %d", results["c1"])
	}

	if results["c2"] != 5 {
		t.Errorf("Expected 5 votes for c2, got %d", results["c2"])
	}

	if results["c3"] != 15 {
		t.Errorf("Expected 15 votes for c3, got %d", results["c3"])
	}
}

func TestEd25519Service_CountVotes_Empty(t *testing.T) {
	service := NewEd25519Service()

	candidates := []Candidate{}
	results := service.CountVotes(candidates)

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
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
