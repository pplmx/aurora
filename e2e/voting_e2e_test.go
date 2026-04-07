package test

import (
	"testing"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	voting "github.com/pplmx/aurora/internal/domain/voting"
)

func TestVotingE2E_CandidateRegistration(t *testing.T) {
	blockchain.ResetForTest()

	cand1 := voting.NewCandidate("Alice", "Party A", "Platform A")
	if cand1.Name != "Alice" {
		t.Errorf("Expected Alice, got %s", cand1.Name)
	}
	if cand1.Party != "Party A" {
		t.Errorf("Expected Party A, got %s", cand1.Party)
	}
	if cand1.VoteCount != 0 {
		t.Errorf("Expected 0 votes, got %d", cand1.VoteCount)
	}
}

func TestVotingE2E_VoterRegistration(t *testing.T) {
	blockchain.ResetForTest()

	voter := voting.NewVoter("Bob")
	if voter.Name != "Bob" {
		t.Errorf("Expected Bob, got %s", voter.Name)
	}
	if voter.HasVoted != false {
		t.Error("New voter should not have voted")
	}
}

func TestVotingE2E_SessionCreation(t *testing.T) {
	blockchain.ResetForTest()

	session := voting.NewSession("Election 2024", "General election", []string{"cand-1", "cand-2"}, 1000, 2000)
	if session.Title != "Election 2024" {
		t.Errorf("Expected Election 2024, got %s", session.Title)
	}
	if session.Status != "draft" {
		t.Errorf("Expected draft status, got %s", session.Status)
	}
	if len(session.Candidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(session.Candidates))
	}
}

func TestVotingE2E_VoteCreation(t *testing.T) {
	blockchain.ResetForTest()

	vote := voting.NewVote("voter-pk", "candidate-id", "signature", "message")
	if vote.VoterPublicKey != "voter-pk" {
		t.Errorf("Expected voter-pk, got %s", vote.VoterPublicKey)
	}
	if vote.CandidateID != "candidate-id" {
		t.Errorf("Expected candidate-id, got %s", vote.CandidateID)
	}
	if vote.Signature != "signature" {
		t.Errorf("Expected signature, got %s", vote.Signature)
	}
	if vote.Timestamp == 0 {
		t.Error("Timestamp should be set")
	}
}
