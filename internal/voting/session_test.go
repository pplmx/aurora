package voting

import (
	"testing"
)

func TestVotingSession_CreateAndGet(t *testing.T) {
	storage := NewInMemoryStorage()
	SetSessionStorage(storage)
	SetCandidateStorage(storage)
	SetVoteStorage(storage)

	candidates := []string{"cand1", "cand2", "cand3"}

	session, err := CreateSession(
		"Test Election",
		"Test election description",
		candidates,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.Title != "Test Election" {
		t.Errorf("Expected title 'Test Election', got '%s'", session.Title)
	}

	if session.Status != "draft" {
		t.Errorf("Expected status 'draft', got '%s'", session.Status)
	}

	retrieved, err := GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetSession returned nil")
	}

	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}
}

func TestVotingSession_ListSessions(t *testing.T) {
	storage := NewInMemoryStorage()
	SetSessionStorage(storage)

	_, err := CreateSession("Election 1", "Desc 1", []string{"c1"}, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	_, err = CreateSession("Election 2", "Desc 2", []string{"c2"}, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestVotingSession_StartAndEnd(t *testing.T) {
	storage := NewInMemoryStorage()
	SetSessionStorage(storage)

	session, err := CreateSession("Election", "Desc", []string{"c1"}, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = StartSession(session.ID)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	updated, _ := GetSession(session.ID)
	if updated.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", updated.Status)
	}

	err = EndSession(session.ID)
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	updated, _ = GetSession(session.ID)
	if updated.Status != "ended" {
		t.Errorf("Expected status 'ended', got '%s'", updated.Status)
	}
}

func TestVotingSession_GetSessionResults(t *testing.T) {
	storage := NewInMemoryStorage()
	SetSessionStorage(storage)
	SetCandidateStorage(storage)
	SetVoteStorage(storage)

	cand := &DBCandidate{ID: "cand1", Name: "Candidate 1", Party: "Party A"}
	storage.SaveCandidate(cand)

	session, err := CreateSession("Election", "Desc", []string{"cand1"}, 0, 0)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	results, err := GetSessionResults(session.ID)
	if err != nil {
		t.Fatalf("GetSessionResults failed: %v", err)
	}

	if results["cand1"] != 0 {
		t.Errorf("Expected 0 votes, got %d", results["cand1"])
	}
}
