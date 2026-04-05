package voting

import (
	"time"

	"github.com/google/uuid"
)

// Candidate is a struct that represents a candidate in the voting system
type Candidate struct {
	ID        string // The unique identifier of the candidate
	Name      string // The name of the candidate
	Party     string // The party of the candidate
	Program   string // The program of the candidate
	Image     string // The image of the candidate
	VoteCount int    // The number of votes received
	CreatedAt int64  // The creation timestamp
}

var candidateStorage Storage

func SetCandidateStorage(s Storage) {
	candidateStorage = s
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

func (c *Candidate) ToDBCandidate() *DBCandidate {
	return &DBCandidate{
		ID:        c.ID,
		Name:      c.Name,
		Party:     c.Party,
		Program:   c.Program,
		ImageURL:  c.Image,
		VoteCount: c.VoteCount,
		CreatedAt: c.CreatedAt,
	}
}

func CandidateFromDBCandidate(db *DBCandidate) *Candidate {
	if db == nil {
		return nil
	}
	return &Candidate{
		ID:        db.ID,
		Name:      db.Name,
		Party:     db.Party,
		Program:   db.Program,
		Image:     db.ImageURL,
		VoteCount: db.VoteCount,
		CreatedAt: db.CreatedAt,
	}
}

func RegisterCandidate(name, party, program string) (*Candidate, error) {
	candidate := NewCandidate(name, party, program)
	if err := candidateStorage.SaveCandidate(candidate.ToDBCandidate()); err != nil {
		return nil, err
	}
	return candidate, nil
}

func GetCandidate(id string) (*Candidate, error) {
	dbCandidate, err := candidateStorage.GetCandidate(id)
	if err != nil {
		return nil, err
	}
	return CandidateFromDBCandidate(dbCandidate), nil
}

func ListCandidates() ([]*Candidate, error) {
	dbCandidates, err := candidateStorage.ListCandidates()
	if err != nil {
		return nil, err
	}
	candidates := make([]*Candidate, len(dbCandidates))
	for i, dbC := range dbCandidates {
		candidates[i] = CandidateFromDBCandidate(dbC)
	}
	return candidates, nil
}

func UpdateCandidate(c *Candidate) error {
	return candidateStorage.UpdateCandidate(c.ToDBCandidate())
}

func DeleteCandidate(id string) error {
	return candidateStorage.DeleteCandidate(id)
}

// GetName is a method that returns the name of the candidate
func (c *Candidate) GetName() string {
	return c.Name
}

// GetParty is a method that returns the party of the candidate
func (c *Candidate) GetParty() string {
	return c.Party
}

// GetProgram is a method that returns the program of the candidate
func (c *Candidate) GetProgram() string {
	return c.Program
}

// GetImage is a method that returns the image of the candidate
func (c *Candidate) GetImage() string {
	return c.Image
}
