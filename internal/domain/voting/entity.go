package voting

import (
	"time"

	"github.com/google/uuid"
)

type Vote struct {
	ID          string
	VoterPK     string
	CandidateID string
	Signature   string
	Message     string
	Timestamp   int64
	BlockHeight int64
}

func NewVote(voterPK, candidateID, signature, message string) *Vote {
	return &Vote{
		ID:          uuid.New().String(),
		VoterPK:     voterPK,
		CandidateID: candidateID,
		Signature:   signature,
		Message:     message,
		Timestamp:   time.Now().Unix(),
		BlockHeight: 0,
	}
}

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
		Status:      "draft",
		Candidates:  candidateIDs,
		CreatedAt:   time.Now().Unix(),
	}
}
