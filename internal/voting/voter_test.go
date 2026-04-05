package voting

import (
	"encoding/base64"
	"testing"
)

func TestVoterRegistration(t *testing.T) {
	storage := NewInMemoryStorage()
	SetVoterStorage(storage)

	publicKey, privateKey, err := RegisterVoter("Alice")
	if err != nil {
		t.Fatalf("RegisterVoter failed: %v", err)
	}

	if len(publicKey) != 32 {
		t.Errorf("expected public key length 32, got %d", len(publicKey))
	}

	if len(privateKey) != 64 {
		t.Errorf("expected private key length 64, got %d", len(privateKey))
	}

	voter, err := GetVoter(base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatalf("GetVoter failed: %v", err)
	}

	if voter.Name != "Alice" {
		t.Errorf("expected voter name 'Alice', got '%s'", voter.Name)
	}

	if voter.HasVoted {
		t.Error("expected HasVoted to be false initially")
	}
}

func TestCanVote(t *testing.T) {
	storage := NewInMemoryStorage()
	SetVoterStorage(storage)

	pub, _, err := RegisterVoter("Bob")
	if err != nil {
		t.Fatalf("RegisterVoter failed: %v", err)
	}

	canVote, err := CanVote(base64.StdEncoding.EncodeToString(pub))
	if err != nil {
		t.Fatalf("CanVote failed: %v", err)
	}
	if !canVote {
		t.Error("expected CanVote to return true for registered voter who hasn't voted")
	}

	canVote, err = CanVote("")
	if err != nil {
		t.Fatalf("CanVote failed: %v", err)
	}
	if canVote {
		t.Error("expected CanVote to return false for empty public key")
	}
}

func TestMarkVoted(t *testing.T) {
	storage := NewInMemoryStorage()
	SetVoterStorage(storage)

	pub, _, err := RegisterVoter("Charlie")
	if err != nil {
		t.Fatalf("RegisterVoter failed: %v", err)
	}

	pubKey := base64.StdEncoding.EncodeToString(pub)
	err = MarkVoted(pubKey, "vote_hash_123")
	if err != nil {
		t.Fatalf("MarkVoted failed: %v", err)
	}

	canVote, err := CanVote(pubKey)
	if err != nil {
		t.Fatalf("CanVote failed: %v", err)
	}
	if canVote {
		t.Error("expected CanVote to return false after MarkVoted")
	}

	voter, err := GetVoter(pubKey)
	if err != nil {
		t.Fatalf("GetVoter failed: %v", err)
	}
	if voter.VoteHash != "vote_hash_123" {
		t.Errorf("expected vote hash 'vote_hash_123', got '%s'", voter.VoteHash)
	}
}

func TestListVoters(t *testing.T) {
	storage := NewInMemoryStorage()
	SetVoterStorage(storage)

	_, _, err := RegisterVoter("Dave")
	if err != nil {
		t.Fatalf("RegisterVoter failed: %v", err)
	}
	_, _, err = RegisterVoter("Eve")
	if err != nil {
		t.Fatalf("RegisterVoter failed: %v", err)
	}

	voters, err := ListVoters()
	if err != nil {
		t.Fatalf("ListVoters failed: %v", err)
	}

	if len(voters) != 2 {
		t.Errorf("expected 2 voters, got %d", len(voters))
	}
}
