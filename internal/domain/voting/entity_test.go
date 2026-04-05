package voting

import (
	"testing"
)

func TestNewVote(t *testing.T) {
	vote := NewVote("voter-pk", "candidate-1", "sig", "msg")

	if vote.ID == "" {
		t.Error("Vote ID should not be empty")
	}
	if vote.VoterPK != "voter-pk" {
		t.Errorf("VoterPK = %v, want voter-pk", vote.VoterPK)
	}
	if vote.CandidateID != "candidate-1" {
		t.Errorf("CandidateID = %v, want candidate-1", vote.CandidateID)
	}
	if vote.Signature != "sig" {
		t.Errorf("Signature = %v, want sig", vote.Signature)
	}
}

func TestNewVoter(t *testing.T) {
	voter := NewVoter("John")

	if voter.Name != "John" {
		t.Errorf("Name = %v, want John", voter.Name)
	}
	if voter.HasVoted != false {
		t.Error("HasVoted should be false")
	}
}

func TestNewCandidate(t *testing.T) {
	cand := NewCandidate("Alice", "Party A", "Test program")

	if cand.Name != "Alice" {
		t.Errorf("Name = %v, want Alice", cand.Name)
	}
	if cand.Party != "Party A" {
		t.Errorf("Party = %v, want Party A", cand.Party)
	}
	if cand.Program != "Test program" {
		t.Errorf("Program = %v, want Test program", cand.Program)
	}
}

func TestNewSession(t *testing.T) {
	session := NewSession("Test Election", "Description", []string{"cand-1"}, 1000, 2000)

	if session.Title != "Test Election" {
		t.Errorf("Title = %v, want Test Election", session.Title)
	}
	if session.Description != "Description" {
		t.Errorf("Description = %v, want Description", session.Description)
	}
	if session.Status != "draft" {
		t.Errorf("Status = %v, want draft", session.Status)
	}
	if len(session.Candidates) != 1 {
		t.Errorf("Candidates length = %v, want 1", len(session.Candidates))
	}
}
