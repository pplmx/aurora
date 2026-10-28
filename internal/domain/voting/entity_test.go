package voting

import (
	"errors"
	"testing"
	"time"
)

func TestNewVote(t *testing.T) {
	vote := NewVote("voter-pk", "candidate-1", "sig", "msg")

	if vote.ID == "" {
		t.Error("Vote ID should not be empty")
	}
	if vote.VoterPublicKey != "voter-pk" {
		t.Errorf("VoterPublicKey = %v, want voter-pk", vote.VoterPublicKey)
	}
	if vote.CandidateID != "candidate-1" {
		t.Errorf("CandidateID = %v, want candidate-1", vote.CandidateID)
	}
	if vote.Signature != "sig" {
		t.Errorf("Signature = %v, want sig", vote.Signature)
	}
}

func TestNewSessionVote(t *testing.T) {
	vote := NewSessionVote("voter-pk", "candidate-1", "session-1", "sig", "msg")
	if vote.SessionID != "session-1" {
		t.Errorf("SessionID = %v, want session-1", vote.SessionID)
	}
	if vote.CandidateID != "candidate-1" {
		t.Errorf("CandidateID = %v, want candidate-1", vote.CandidateID)
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

func TestSessionCanAcceptVotes(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	tests := []struct {
		name    string
		status  string
		start   int64
		end     int64
		wantErr error
	}{
		{
			name:    "draft inside window",
			status:  SessionStatusDraft,
			start:   now.Unix() - 60,
			end:     now.Unix() + 60,
			wantErr: ErrSessionNotStarted,
		},
		{
			name:   "active inside window",
			status: SessionStatusActive,
			start:  now.Unix() - 60,
			end:    now.Unix() + 60,
		},
		{
			name:    "active before start",
			status:  SessionStatusActive,
			start:   now.Unix() + 1,
			end:     now.Unix() + 60,
			wantErr: ErrSessionNotStarted,
		},
		{
			name:    "active after end",
			status:  SessionStatusActive,
			start:   now.Unix() - 60,
			end:     now.Unix() - 1,
			wantErr: ErrSessionEnded,
		},
		{
			name:    "ended inside window",
			status:  SessionStatusEnded,
			start:   now.Unix() - 60,
			end:     now.Unix() + 60,
			wantErr: ErrSessionEnded,
		},
		{
			name:    "unknown status fails closed",
			status:  "paused",
			start:   now.Unix() - 60,
			end:     now.Unix() + 60,
			wantErr: ErrSessionNotStarted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				Status:    tt.status,
				StartTime: tt.start,
				EndTime:   tt.end,
			}
			if err := session.CanAcceptVotes(now); !errors.Is(err, tt.wantErr) {
				t.Fatalf("CanAcceptVotes() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
