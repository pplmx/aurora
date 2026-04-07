package sqlite

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/voting"
)

func setupVotingTestDB(t *testing.T) (*VotingRepository, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS votes (
			id TEXT PRIMARY KEY,
			voter_pk TEXT NOT NULL,
			candidate_id TEXT NOT NULL,
			signature TEXT,
			message TEXT,
			timestamp INTEGER,
			block_height INTEGER
		);
		CREATE TABLE IF NOT EXISTS voters (
			public_key TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			has_voted INTEGER DEFAULT 0,
			vote_hash TEXT,
			registered_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS candidates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			party TEXT,
			program TEXT,
			description TEXT,
			image_url TEXT,
			vote_count INTEGER DEFAULT 0,
			created_at INTEGER
		);
		CREATE TABLE IF NOT EXISTS voting_sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			start_time INTEGER,
			end_time INTEGER,
			status TEXT,
			candidates TEXT,
			created_at INTEGER
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	repo := NewVotingRepository(db)

	cleanup := func() {
		db.Close()
		os.RemoveAll("./data")
	}

	return repo, cleanup
}

func TestNewVotingRepository(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewVotingRepository(db)
	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
	if repo.db == nil {
		t.Fatal("Database should not be nil")
	}
}

func TestVotingRepository_SaveVoter(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	voter := voting.NewVoter("John Doe")

	err := repo.SaveVoter(voter)
	if err != nil {
		t.Fatalf("Failed to save voter: %v", err)
	}
}

func TestVotingRepository_GetVoter(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	voter := voting.NewVoter("John Doe")
	voter.PublicKey = "test-pk"

	err := repo.SaveVoter(voter)
	if err != nil {
		t.Fatalf("Failed to save voter: %v", err)
	}

	retrieved, err := repo.GetVoter("test-pk")
	if err != nil {
		t.Fatalf("Failed to get voter: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Voter should not be nil")
	}

	if retrieved.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", retrieved.Name)
	}
}

func TestVotingRepository_GetVoter_NotFound(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	_, err := repo.GetVoter("NOTEXIST")
	if err != nil {
		t.Fatalf("Expected nil for non-existent voter, got error: %v", err)
	}
}

func TestVotingRepository_SaveCandidate(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	candidate := voting.NewCandidate("Alice", "Party A", "Platform A")

	err := repo.SaveCandidate(candidate)
	if err != nil {
		t.Fatalf("Failed to save candidate: %v", err)
	}
}

func TestVotingRepository_GetCandidate(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	candidate := voting.NewCandidate("Alice", "Party A", "Platform A")

	err := repo.SaveCandidate(candidate)
	if err != nil {
		t.Fatalf("Failed to save candidate: %v", err)
	}

	retrieved, err := repo.GetCandidate(candidate.ID)
	if err != nil {
		t.Fatalf("Failed to get candidate: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Candidate should not be nil")
	}

	if retrieved.Name != "Alice" {
		t.Errorf("Expected name 'Alice', got '%s'", retrieved.Name)
	}
}

func TestVotingRepository_ListCandidates(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	cand1 := voting.NewCandidate("Alice", "Party A", "Platform A")
	cand2 := voting.NewCandidate("Bob", "Party B", "Platform B")

	err := repo.SaveCandidate(cand1)
	if err != nil {
		t.Fatalf("Failed to save cand1: %v", err)
	}
	err = repo.SaveCandidate(cand2)
	if err != nil {
		t.Fatalf("Failed to save cand2: %v", err)
	}

	candidates, err := repo.ListCandidates()
	if err != nil {
		t.Fatalf("Failed to list candidates: %v", err)
	}

	if len(candidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(candidates))
	}
}

func TestVotingRepository_SaveVote(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	vote := voting.NewVote("voter-pk", "candidate-id", "signature", "message")

	err := repo.SaveVote(vote)
	if err != nil {
		t.Fatalf("Failed to save vote: %v", err)
	}
}

func TestVotingRepository_GetVote(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	vote := voting.NewVote("voter-pk", "candidate-id", "signature", "message")

	err := repo.SaveVote(vote)
	if err != nil {
		t.Fatalf("Failed to save vote: %v", err)
	}

	retrieved, err := repo.GetVote(vote.ID)
	if err != nil {
		t.Fatalf("Failed to get vote: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Vote should not be nil")
	}

	if retrieved.CandidateID != "candidate-id" {
		t.Errorf("Expected candidate ID 'candidate-id', got '%s'", retrieved.CandidateID)
	}
}

func TestVotingRepository_GetVotesByCandidate(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	vote1 := voting.NewVote("voter1", "cand-1", "sig1", "msg1")
	vote2 := voting.NewVote("voter2", "cand-1", "sig2", "msg2")
	vote3 := voting.NewVote("voter3", "cand-2", "sig3", "msg3")

	err := repo.SaveVote(vote1)
	if err != nil {
		t.Fatalf("Failed to save vote1: %v", err)
	}
	err = repo.SaveVote(vote2)
	if err != nil {
		t.Fatalf("Failed to save vote2: %v", err)
	}
	err = repo.SaveVote(vote3)
	if err != nil {
		t.Fatalf("Failed to save vote3: %v", err)
	}

	votes, err := repo.GetVotesByCandidate("cand-1")
	if err != nil {
		t.Fatalf("Failed to get votes: %v", err)
	}

	if len(votes) != 2 {
		t.Errorf("Expected 2 votes, got %d", len(votes))
	}
}

func TestVotingRepository_SaveSession(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	session := voting.NewSession("Election 2024", "General election", []string{"cand-1", "cand-2"}, 1000, 2000)

	err := repo.SaveSession(session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}
}

func TestVotingRepository_GetSession(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	session := voting.NewSession("Election 2024", "General election", []string{"cand-1", "cand-2"}, 1000, 2000)

	err := repo.SaveSession(session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	retrieved, err := repo.GetSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Session should not be nil")
	}

	if retrieved.Title != "Election 2024" {
		t.Errorf("Expected title 'Election 2024', got '%s'", retrieved.Title)
	}
}

func TestVotingRepository_UpdateSession(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	session := voting.NewSession("Election 2024", "General election", []string{"cand-1"}, 1000, 2000)

	err := repo.SaveSession(session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	session.Status = "active"
	err = repo.UpdateSession(session)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	retrieved, err := repo.GetSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", retrieved.Status)
	}
}
