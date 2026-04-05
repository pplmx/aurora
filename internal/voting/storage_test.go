package voting

import (
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteStorage(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "voting_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	storage, err := NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	t.Run("Candidate CRUD", func(t *testing.T) {
		candidate := &DBCandidate{
			Name:        "John Doe",
			Party:       "Democratic",
			Program:     "Test Program",
			Description: "Test Description",
			ImageURL:    "https://example.com/image.jpg",
			VoteCount:   0,
			CreatedAt:   time.Now().Unix(),
		}

		if err := storage.SaveCandidate(candidate); err != nil {
			t.Fatalf("failed to save candidate: %v", err)
		}

		if candidate.ID == "" {
			t.Fatal("candidate ID should be generated")
		}

		fetched, err := storage.GetCandidate(candidate.ID)
		if err != nil {
			t.Fatalf("failed to get candidate: %v", err)
		}
		if fetched == nil {
			t.Fatal("candidate should be found")
		}
		if fetched.Name != candidate.Name {
			t.Errorf("expected name %s, got %s", candidate.Name, fetched.Name)
		}

		candidate.VoteCount = 10
		if err := storage.UpdateCandidate(candidate); err != nil {
			t.Fatalf("failed to update candidate: %v", err)
		}

		fetched, _ = storage.GetCandidate(candidate.ID)
		if fetched.VoteCount != 10 {
			t.Errorf("expected vote count 10, got %d", fetched.VoteCount)
		}

		if err := storage.DeleteCandidate(candidate.ID); err != nil {
			t.Fatalf("failed to delete candidate: %v", err)
		}

		fetched, _ = storage.GetCandidate(candidate.ID)
		if fetched != nil {
			t.Error("candidate should be deleted")
		}
	})

	t.Run("Voter CRUD", func(t *testing.T) {
		voter := &DBVoter{
			PublicKey:    "test-public-key-123",
			Name:         "Jane Doe",
			HasVoted:     false,
			VoteHash:     "",
			RegisteredAt: time.Now().Unix(),
		}

		if err := storage.SaveVoter(voter); err != nil {
			t.Fatalf("failed to save voter: %v", err)
		}

		fetched, err := storage.GetVoter(voter.PublicKey)
		if err != nil {
			t.Fatalf("failed to get voter: %v", err)
		}
		if fetched == nil {
			t.Fatal("voter should be found")
		}
		if fetched.Name != voter.Name {
			t.Errorf("expected name %s, got %s", voter.Name, fetched.Name)
		}

		voter.HasVoted = true
		voter.VoteHash = "abc123"
		if err := storage.UpdateVoter(voter); err != nil {
			t.Fatalf("failed to update voter: %v", err)
		}

		fetched, _ = storage.GetVoter(voter.PublicKey)
		if !fetched.HasVoted {
			t.Error("voter should have voted")
		}
		if fetched.VoteHash != "abc123" {
			t.Errorf("expected vote hash abc123, got %s", fetched.VoteHash)
		}
	})

	t.Run("Vote Record CRUD", func(t *testing.T) {
		candidate := &DBCandidate{
			Name:      "Candidate 1",
			Party:     "Party A",
			Program:   "Program",
			CreatedAt: time.Now().Unix(),
		}
		storage.SaveCandidate(candidate)

		voter := &DBVoter{
			PublicKey:    "voter-pk-123",
			Name:         "Voter 1",
			RegisteredAt: time.Now().Unix(),
		}
		storage.SaveVoter(voter)

		vote := &DBVoteRecord{
			VoterPK:     voter.PublicKey,
			CandidateID: candidate.ID,
			Signature:   "signature-abc",
			Message:     "vote message",
			Timestamp:   time.Now().Unix(),
			BlockHeight: 100,
		}

		if err := storage.SaveVote(vote); err != nil {
			t.Fatalf("failed to save vote: %v", err)
		}

		if vote.ID == "" {
			t.Fatal("vote ID should be generated")
		}

		fetched, err := storage.GetVote(vote.ID)
		if err != nil {
			t.Fatalf("failed to get vote: %v", err)
		}
		if fetched == nil {
			t.Fatal("vote should be found")
		}
		if fetched.Signature != vote.Signature {
			t.Errorf("expected signature %s, got %s", vote.Signature, fetched.Signature)
		}

		votes, err := storage.GetVotesByCandidate(candidate.ID)
		if err != nil {
			t.Fatalf("failed to get votes by candidate: %v", err)
		}
		if len(votes) != 1 {
			t.Errorf("expected 1 vote, got %d", len(votes))
		}

		votes, err = storage.GetVotesByVoter(voter.PublicKey)
		if err != nil {
			t.Fatalf("failed to get votes by voter: %v", err)
		}
		if len(votes) != 1 {
			t.Errorf("expected 1 vote, got %d", len(votes))
		}
	})

	t.Run("Voting Session CRUD", func(t *testing.T) {
		session := &DBVotingSession{
			Title:       "Election 2024",
			Description: "Presidential Election",
			StartTime:   time.Now().Unix(),
			EndTime:     time.Now().Add(24 * time.Hour).Unix(),
			Status:      "active",
			Candidates:  []string{"cand1", "cand2"},
			CreatedAt:   time.Now().Unix(),
		}

		if err := storage.SaveSession(session); err != nil {
			t.Fatalf("failed to save session: %v", err)
		}

		if session.ID == "" {
			t.Fatal("session ID should be generated")
		}

		fetched, err := storage.GetSession(session.ID)
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}
		if fetched == nil {
			t.Fatal("session should be found")
		}
		if fetched.Title != session.Title {
			t.Errorf("expected title %s, got %s", session.Title, fetched.Title)
		}
		if len(fetched.Candidates) != 2 {
			t.Errorf("expected 2 candidates, got %d", len(fetched.Candidates))
		}

		session.Status = "closed"
		if err := storage.UpdateSession(session); err != nil {
			t.Fatalf("failed to update session: %v", err)
		}

		fetched, _ = storage.GetSession(session.ID)
		if fetched.Status != "closed" {
			t.Errorf("expected status closed, got %s", fetched.Status)
		}
	})

	t.Run("List operations", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			candidate := &DBCandidate{
				Name:      "Candidate " + string(rune('A'+i)),
				Party:     "Party",
				Program:   "Program",
				CreatedAt: time.Now().Unix(),
			}
			storage.SaveCandidate(candidate)
		}

		candidates, err := storage.ListCandidates()
		if err != nil {
			t.Fatalf("failed to list candidates: %v", err)
		}
		if len(candidates) < 3 {
			t.Errorf("expected at least 3 candidates, got %d", len(candidates))
		}
	})
}

func TestInMemoryStorage(t *testing.T) {
	storage := NewInMemoryStorage()

	t.Run("Candidate CRUD", func(t *testing.T) {
		candidate := &DBCandidate{
			Name:      "John Doe",
			Party:     "Democratic",
			Program:   "Test Program",
			VoteCount: 0,
			CreatedAt: time.Now().Unix(),
		}

		if err := storage.SaveCandidate(candidate); err != nil {
			t.Fatalf("failed to save candidate: %v", err)
		}

		if candidate.ID == "" {
			t.Fatal("candidate ID should be generated")
		}

		fetched, err := storage.GetCandidate(candidate.ID)
		if err != nil {
			t.Fatalf("failed to get candidate: %v", err)
		}
		if fetched == nil {
			t.Fatal("candidate should be found")
		}
		if fetched.Name != candidate.Name {
			t.Errorf("expected name %s, got %s", candidate.Name, fetched.Name)
		}

		candidate.VoteCount = 10
		if err := storage.UpdateCandidate(candidate); err != nil {
			t.Fatalf("failed to update candidate: %v", err)
		}

		if err := storage.DeleteCandidate(candidate.ID); err != nil {
			t.Fatalf("failed to delete candidate: %v", err)
		}

		fetched, _ = storage.GetCandidate(candidate.ID)
		if fetched != nil {
			t.Error("candidate should be deleted")
		}
	})

	t.Run("Voter CRUD", func(t *testing.T) {
		voter := &DBVoter{
			PublicKey:    "test-public-key",
			Name:         "Jane Doe",
			HasVoted:     false,
			RegisteredAt: time.Now().Unix(),
		}

		if err := storage.SaveVoter(voter); err != nil {
			t.Fatalf("failed to save voter: %v", err)
		}

		fetched, err := storage.GetVoter(voter.PublicKey)
		if err != nil {
			t.Fatalf("failed to get voter: %v", err)
		}
		if fetched == nil {
			t.Fatal("voter should be found")
		}

		voter.HasVoted = true
		if err := storage.UpdateVoter(voter); err != nil {
			t.Fatalf("failed to update voter: %v", err)
		}
	})

	t.Run("Vote Record CRUD", func(t *testing.T) {
		candidate := &DBCandidate{
			Name:      "Candidate 1",
			Party:     "Party A",
			Program:   "Program",
			CreatedAt: time.Now().Unix(),
		}
		storage.SaveCandidate(candidate)

		voter := &DBVoter{
			PublicKey:    "voter-pk",
			Name:         "Voter 1",
			RegisteredAt: time.Now().Unix(),
		}
		storage.SaveVoter(voter)

		vote := &DBVoteRecord{
			VoterPK:     voter.PublicKey,
			CandidateID: candidate.ID,
			Signature:   "sig123",
			Timestamp:   time.Now().Unix(),
		}

		if err := storage.SaveVote(vote); err != nil {
			t.Fatalf("failed to save vote: %v", err)
		}

		fetched, err := storage.GetVote(vote.ID)
		if err != nil {
			t.Fatalf("failed to get vote: %v", err)
		}
		if fetched == nil {
			t.Fatal("vote should be found")
		}

		votes, _ := storage.GetVotesByCandidate(candidate.ID)
		if len(votes) != 1 {
			t.Errorf("expected 1 vote, got %d", len(votes))
		}
	})

	t.Run("Voting Session CRUD", func(t *testing.T) {
		session := &DBVotingSession{
			Title:      "Test Election",
			StartTime:  time.Now().Unix(),
			EndTime:    time.Now().Add(24 * time.Hour).Unix(),
			Status:     "active",
			Candidates: []string{"c1", "c2"},
			CreatedAt:  time.Now().Unix(),
		}

		if err := storage.SaveSession(session); err != nil {
			t.Fatalf("failed to save session: %v", err)
		}

		if session.ID == "" {
			t.Fatal("session ID should be generated")
		}

		fetched, err := storage.GetSession(session.ID)
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}
		if fetched == nil {
			t.Fatal("session should be found")
		}
		if fetched.Title != session.Title {
			t.Errorf("expected title %s, got %s", session.Title, fetched.Title)
		}
	})

	t.Run("Transaction methods", func(t *testing.T) {
		if err := storage.Begin(); err != nil {
			t.Fatalf("failed to begin: %v", err)
		}
		if err := storage.Commit(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}
		if err := storage.Rollback(); err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}
		if err := storage.Close(); err != nil {
			t.Fatalf("failed to close: %v", err)
		}
	})
}
