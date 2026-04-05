package voting

import (
	"testing"
)

func TestCandidateManagement(t *testing.T) {
	storage := NewInMemoryStorage()
	SetCandidateStorage(storage)

	candidate, err := RegisterCandidate("John Doe", "Party A", "Program 1")
	if err != nil {
		t.Fatalf("Failed to register candidate: %v", err)
	}
	if candidate == nil {
		t.Fatal("Candidate is nil")
	}
	if candidate.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", candidate.Name)
	}
	if candidate.Party != "Party A" {
		t.Errorf("Expected party 'Party A', got '%s'", candidate.Party)
	}
	if candidate.Program != "Program 1" {
		t.Errorf("Expected program 'Program 1', got '%s'", candidate.Program)
	}
	if candidate.VoteCount != 0 {
		t.Errorf("Expected VoteCount 0, got %d", candidate.VoteCount)
	}
	if candidate.ID == "" {
		t.Error("Candidate ID should not be empty")
	}

	retrieved, err := GetCandidate(candidate.ID)
	if err != nil {
		t.Fatalf("Failed to get candidate: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved candidate is nil")
	}
	if retrieved.ID != candidate.ID {
		t.Errorf("Expected ID '%s', got '%s'", candidate.ID, retrieved.ID)
	}
	if retrieved.Name != candidate.Name {
		t.Errorf("Expected name '%s', got '%s'", candidate.Name, retrieved.Name)
	}

	candidates, err := ListCandidates()
	if err != nil {
		t.Fatalf("Failed to list candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Errorf("Expected 1 candidate, got %d", len(candidates))
	}

	candidate2, _ := RegisterCandidate("Jane Doe", "Party B", "Program 2")
	candidates, err = ListCandidates()
	if err != nil {
		t.Fatalf("Failed to list candidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(candidates))
	}

	err = DeleteCandidate(candidate2.ID)
	if err != nil {
		t.Fatalf("Failed to delete candidate: %v", err)
	}

	candidates, err = ListCandidates()
	if err != nil {
		t.Fatalf("Failed to list candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Errorf("Expected 1 candidate after delete, got %d", len(candidates))
	}

	candidate.VoteCount = 10
	err = UpdateCandidate(candidate)
	if err != nil {
		t.Fatalf("Failed to update candidate: %v", err)
	}

	updated, _ := GetCandidate(candidate.ID)
	if updated.VoteCount != 10 {
		t.Errorf("Expected VoteCount 10, got %d", updated.VoteCount)
	}
}

func TestNewCandidate(t *testing.T) {
	candidate := NewCandidate("Test Candidate", "Test Party", "Test Program")
	if candidate == nil {
		t.Fatal("NewCandidate returned nil")
	}
	if candidate.Name != "Test Candidate" {
		t.Errorf("Expected name 'Test Candidate', got '%s'", candidate.Name)
	}
	if candidate.Party != "Test Party" {
		t.Errorf("Expected party 'Test Party', got '%s'", candidate.Party)
	}
	if candidate.Program != "Test Program" {
		t.Errorf("Expected program 'Test Program', got '%s'", candidate.Program)
	}
	if candidate.ID == "" {
		t.Error("ID should not be empty")
	}
	if candidate.VoteCount != 0 {
		t.Errorf("Expected VoteCount 0, got %d", candidate.VoteCount)
	}
	if candidate.CreatedAt == 0 {
		t.Error("CreatedAt should not be zero")
	}
}

func TestCandidateConversion(t *testing.T) {
	dbCandidate := &DBCandidate{
		ID:        "test-id",
		Name:      "Test Name",
		Party:     "Test Party",
		Program:   "Test Program",
		ImageURL:  "http://example.com/image.jpg",
		VoteCount: 5,
		CreatedAt: 1234567890,
	}

	candidate := CandidateFromDBCandidate(dbCandidate)
	if candidate.ID != dbCandidate.ID {
		t.Errorf("Expected ID '%s', got '%s'", dbCandidate.ID, candidate.ID)
	}
	if candidate.Name != dbCandidate.Name {
		t.Errorf("Expected name '%s', got '%s'", dbCandidate.Name, candidate.Name)
	}
	if candidate.Image != dbCandidate.ImageURL {
		t.Errorf("Expected Image '%s', got '%s'", dbCandidate.ImageURL, candidate.Image)
	}

	dbBack := candidate.ToDBCandidate()
	if dbBack.ID != candidate.ID {
		t.Errorf("Expected ID '%s', got '%s'", candidate.ID, dbBack.ID)
	}
	if dbBack.ImageURL != candidate.Image {
		t.Errorf("Expected ImageURL '%s', got '%s'", candidate.Image, dbBack.ImageURL)
	}

	nilCandidate := CandidateFromDBCandidate(nil)
	if nilCandidate != nil {
		t.Error("Expected nil for nil DBCandidate")
	}
}
