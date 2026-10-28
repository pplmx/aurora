// Package voting provides transparent voting system with Ed25519 signature
// verification and blockchain-based vote recording.
package voting

import (
	"time"

	"github.com/google/uuid"
)

// Voting session lifecycle states. Only an explicitly active session may
// accept ballots; draft and unknown states fail closed.
const (
	SessionStatusDraft  = "draft"
	SessionStatusActive = "active"
	SessionStatusEnded  = "ended"
)

// Vote represents a single vote in the voting system.
type Vote struct {
	ID             string
	VoterPublicKey string
	CandidateID    string
	SessionID      string
	Signature      string
	Message        string
	Timestamp      int64
	BlockHeight    int64
}

func NewVote(voterPublicKey, candidateID, signature, message string) *Vote {
	return &Vote{
		ID:             uuid.New().String(),
		VoterPublicKey: voterPublicKey,
		CandidateID:    candidateID,
		Signature:      signature,
		Message:        message,
		Timestamp:      time.Now().Unix(),
		BlockHeight:    0,
	}
}

// NewSessionVote creates a vote bound to the election session it was cast in.
// SessionID is what makes per-session results possible when a candidate is
// reused across elections.
func NewSessionVote(voterPublicKey, candidateID, sessionID, signature, message string) *Vote {
	vote := NewVote(voterPublicKey, candidateID, signature, message)
	vote.SessionID = sessionID
	return vote
}

// Voter represents a voter in the voting system.
type Voter struct {
	PublicKey    string
	Name         string
	HasVoted     bool
	VoteHash     string
	RegisteredAt int64
}

func NewVoter(name string) *Voter {
	return &Voter{
		Name:         name,
		HasVoted:     false,
		RegisteredAt: time.Now().Unix(),
	}
}

// Candidate represents a candidate in an election.
type Candidate struct {
	ID        string
	Name      string
	Party     string
	Program   string
	ImageURL  string
	VoteCount int
	CreatedAt int64
}

func NewCandidate(name, party, program string) *Candidate {
	return &Candidate{
		ID:        uuid.New().String(),
		Name:      name,
		Party:     party,
		Program:   program,
		VoteCount: 0,
		CreatedAt: time.Now().Unix(),
	}
}

// Session represents a voting session/election.
type Session struct {
	ID          string
	Title       string
	Description string
	StartTime   int64
	EndTime     int64
	Status      string
	Candidates  []string
	CreatedAt   int64
}

func NewSession(title, description string, candidateIDs []string, startTime, endTime int64) *Session {
	return &Session{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      SessionStatusDraft,
		Candidates:  candidateIDs,
		CreatedAt:   time.Now().Unix(),
	}
}

// CanAcceptVotes enforces both the lifecycle state and the configured ballot
// window. Time alone is not enough: a newly created session is a draft, and an
// operator can end one before its nominal end time.
func (s *Session) CanAcceptVotes(now time.Time) error {
	if s == nil {
		return ErrSessionNotFound
	}
	if s.Status == SessionStatusEnded {
		return ErrSessionEnded
	}
	if now.Unix() < s.StartTime {
		return ErrSessionNotStarted
	}
	if now.Unix() > s.EndTime {
		return ErrSessionEnded
	}
	if s.Status != SessionStatusActive {
		return ErrSessionNotStarted
	}
	return nil
}
