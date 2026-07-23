package sqlite

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/require"
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
		_ = db.Close()
		_ = os.RemoveAll("./data")
	}

	return repo, cleanup
}

func TestNewVotingRepository(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	repo := NewVotingRepository(db)
	if repo == nil {
		t.Fatal("Repository should not be nil")
	} else if repo.db == nil {
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
	if retrieved.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", retrieved.Name)
	}
}

func TestVotingRepository_GetVoter_NotFound(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	_, err := repo.GetVoter("NOTEXIST")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected ErrNotFound, got: %v", err)
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

func TestVotingRepository_DeleteVote(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	vote := voting.NewVote("voter-1", "cand-1", "sig", "msg")
	require.NoError(t, repo.SaveVote(vote))

	require.NoError(t, repo.DeleteVote(vote.ID))

	_, err := repo.GetVote(vote.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestVotingRepository_DeleteVote_AlreadyDeleted(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	vote := voting.NewVote("voter-1", "cand-1", "sig", "msg")
	require.NoError(t, repo.SaveVote(vote))

	require.NoError(t, repo.DeleteVote(vote.ID))
	require.NoError(t, repo.DeleteVote(vote.ID))
}

func TestVotingRepository_GetVotesByVoter(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	require.NoError(t, repo.SaveVote(&voting.Vote{
		ID: "v1", VoterPublicKey: "voter-1", CandidateID: "cand-1", Timestamp: 1000,
	}))
	require.NoError(t, repo.SaveVote(&voting.Vote{
		ID: "v2", VoterPublicKey: "voter-1", CandidateID: "cand-2", Timestamp: 2000,
	}))
	require.NoError(t, repo.SaveVote(&voting.Vote{
		ID: "v3", VoterPublicKey: "voter-2", CandidateID: "cand-1", Timestamp: 1500,
	}))

	votes, err := repo.GetVotesByVoter("voter-1")
	require.NoError(t, err)
	require.Len(t, votes, 2)
	require.Equal(t, "v2", votes[0].ID, "should be sorted by timestamp DESC")
	require.Equal(t, "v1", votes[1].ID)

	votes, err = repo.GetVotesByVoter("voter-2")
	require.NoError(t, err)
	require.Len(t, votes, 1)
	require.Equal(t, "v3", votes[0].ID)

	votes, err = repo.GetVotesByVoter("nonexistent")
	require.NoError(t, err)
	require.Empty(t, votes)
}

func TestVotingRepository_UpdateVoter(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	voter := &voting.Voter{
		PublicKey:    "pk-1",
		Name:         "Original",
		HasVoted:     false,
		VoteHash:     "",
		RegisteredAt: 1000,
	}
	require.NoError(t, repo.SaveVoter(voter))

	voter.Name = "Updated"
	voter.HasVoted = true
	voter.VoteHash = "hash-123"
	require.NoError(t, repo.UpdateVoter(voter))

	retrieved, err := repo.GetVoter("pk-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "Updated", retrieved.Name)
	require.True(t, retrieved.HasVoted)
	require.Equal(t, "hash-123", retrieved.VoteHash)
}

func TestVotingRepository_ListVoters(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-1", Name: "Voter 1", RegisteredAt: 100}))
	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-2", Name: "Voter 2", RegisteredAt: 200}))
	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-3", Name: "Voter 3", RegisteredAt: 150}))

	voters, err := repo.ListVoters()
	require.NoError(t, err)
	require.Len(t, voters, 3)
	require.Equal(t, "Voter 2", voters[0].Name, "should be sorted by registered_at DESC")
	require.Equal(t, "Voter 3", voters[1].Name)
	require.Equal(t, "Voter 1", voters[2].Name)
}

func TestVotingRepository_ListVoters_Empty(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	voters, err := repo.ListVoters()
	require.NoError(t, err)
	require.Empty(t, voters)
}

func TestVotingRepository_UpdateCandidate(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	cand := &voting.Candidate{
		ID:        "cand-1",
		Name:      "Original",
		Party:     "Party A",
		Program:   "Original Program",
		ImageURL:  "https://example.com/orig.png",
		VoteCount: 10,
		CreatedAt: 1000,
	}
	require.NoError(t, repo.SaveCandidate(cand))

	cand.Name = "Updated"
	cand.Party = "Party B"
	cand.Program = "New Program"
	cand.ImageURL = "https://example.com/new.png"
	cand.VoteCount = 20
	require.NoError(t, repo.UpdateCandidate(cand))

	retrieved, err := repo.GetCandidate("cand-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "Updated", retrieved.Name)
	require.Equal(t, "Party B", retrieved.Party)
	require.Equal(t, "New Program", retrieved.Program)
	require.Equal(t, "https://example.com/new.png", retrieved.ImageURL)
	require.Equal(t, 20, retrieved.VoteCount)
}

func TestVotingRepository_DeleteCandidate(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	cand := &voting.Candidate{
		ID:        "cand-1",
		Name:      "ToDelete",
		Party:     "Party A",
		CreatedAt: 1000,
	}
	require.NoError(t, repo.SaveCandidate(cand))

	require.NoError(t, repo.DeleteCandidate("cand-1"))

	_, err := repo.GetCandidate("cand-1")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestVotingRepository_DeleteCandidate_AlreadyDeleted(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	require.NoError(t, repo.DeleteCandidate("nonexistent"))
}

func TestVotingRepository_ListSessions(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	s1 := voting.NewSession("Session 1", "Desc 1", []string{"c1"}, 1000, 2000)
	s1.CreatedAt = 100
	require.NoError(t, repo.SaveSession(s1))

	s2 := voting.NewSession("Session 2", "Desc 2", []string{"c2"}, 3000, 4000)
	s2.CreatedAt = 200
	require.NoError(t, repo.SaveSession(s2))

	s3 := voting.NewSession("Session 3", "Desc 3", []string{"c3"}, 5000, 6000)
	s3.CreatedAt = 150
	require.NoError(t, repo.SaveSession(s3))

	sessions, err := repo.ListSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 3)
	require.Equal(t, "Session 2", sessions[0].Title, "should be sorted by created_at DESC")
	require.Equal(t, "Session 3", sessions[1].Title)
	require.Equal(t, "Session 1", sessions[2].Title)
}

func TestVotingRepository_ListSessions_Empty(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	sessions, err := repo.ListSessions()
	require.NoError(t, err)
	require.Empty(t, sessions)
}
