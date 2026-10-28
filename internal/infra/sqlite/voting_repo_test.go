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
	db, err := sql.Open("sqlite", ":memory:")
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

	repo, err := NewVotingRepository(db)
	if err != nil {
		t.Fatalf("Failed to create voting repository: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll("./data")
	}

	return repo, cleanup
}

func TestNewVotingRepository(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	repo, err := NewVotingRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if repo == nil {
		t.Fatal("Repository should not be nil")
	} else if repo.db == nil {
		t.Fatal("Database should not be nil")
	}
}

// TestNewVotingRepository_SelfBootstrapsSchema pins the bootstrap contract
// (TASK-197): the constructor must create the voting tables on a totally
// empty DB. Every sibling repo (lottery/token/nft/oracle) self-creates its
// schema, and a fresh install that never ran `aurora migrate up` would
// otherwise 500 "no such table" on every voting operation while the other
// modules work out of the box. A write+read through the freshly constructed
// repo proves the tables are not just created but usable; the single pooled
// connection avoids SQLite's per-connection :memory: isolation.
func TestNewVotingRepository_SelfBootstrapsSchema(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// Pin one connection so :memory: does not split into per-connection DBs.
	db.SetMaxOpenConns(1)
	defer func() { _ = db.Close() }()

	repo, err := NewVotingRepository(db)
	require.NoError(t, err)

	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-fresh", Name: "Ada", RegisteredAt: 1}))
	got, err := repo.GetVoter("pk-fresh")
	require.NoError(t, err)
	require.Equal(t, "Ada", got.Name)

	require.NoError(t, repo.SaveCandidate(&voting.Candidate{ID: "cand-fresh", Name: "C", Party: "P", Program: "G", VoteCount: 0, CreatedAt: 1}))
	c, err := repo.GetCandidate("cand-fresh")
	require.NoError(t, err)
	require.Equal(t, "C", c.Name)

	require.NoError(t, repo.SaveSession(&voting.Session{ID: "sess-fresh", Title: "S", Description: "D", StartTime: 1, EndTime: 2, Status: "draft", CreatedAt: 1}))
	s, err := repo.GetSession("sess-fresh")
	require.NoError(t, err)
	require.Equal(t, "draft", s.Status)
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

	candidates, err := repo.ListCandidates(0, 0)
	if err != nil {
		t.Fatalf("Failed to list candidates: %v", err)
	}

	if len(candidates) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(candidates))
	}
}

func TestVotingRepository_ListCandidatesPages(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	for i, name := range []string{"Alice", "Bob", "Carol"} {
		candidate := voting.NewCandidate(name, "Party", "Program")
		candidate.CreatedAt = int64(100 + i*10)
		require.NoError(t, repo.SaveCandidate(candidate))
	}

	page, err := repo.ListCandidates(1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "Bob", page[0].Name, "the second row in newest-first order")

	past, err := repo.ListCandidates(10, 99)
	require.NoError(t, err)
	require.Empty(t, past, "an offset past the end is an empty page")
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

func TestVotingRepository_GetVotesByCandidate(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	for _, pk := range []string{"voter1", "voter2", "voter3"} {
		require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: pk, Name: pk, RegisteredAt: 1}))
	}
	vote1 := voting.NewSessionVote("voter1", "cand-1", "s1", "sig1", "msg1")
	vote2 := voting.NewSessionVote("voter2", "cand-1", "s1", "sig2", "msg2")
	vote3 := voting.NewSessionVote("voter3", "cand-2", "s1", "sig3", "msg3")

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
	if votes[0].SessionID != "s1" || votes[1].SessionID != "s1" {
		t.Errorf("session_id must round-trip, got %q and %q", votes[0].SessionID, votes[1].SessionID)
	}
}

// TestVotingRepository_CountVotesBySession pins the ISS-299 source of truth:
// per-session tallies count vote rows scoped to that session, so the same
// candidate's global lifetime vote_count cannot leak into a later election.
func TestVotingRepository_CountVotesBySession(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	for _, pk := range []string{"v1", "v2", "v3", "v4"} {
		require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: pk, Name: pk, RegisteredAt: 1}))
	}
	for _, v := range []*voting.Vote{
		voting.NewSessionVote("v1", "cand-1", "s1", "sig", "msg"),
		voting.NewSessionVote("v2", "cand-1", "s1", "sig", "msg"),
		voting.NewSessionVote("v3", "cand-1", "s2", "sig", "msg"),
		voting.NewSessionVote("v4", "cand-2", "s1", "sig", "msg"),
	} {
		require.NoError(t, repo.SaveVote(v))
	}

	n, err := repo.CountVotesBySession("s1", "cand-1")
	require.NoError(t, err)
	require.Equal(t, 2, n)

	n, err = repo.CountVotesBySession("s2", "cand-1")
	require.NoError(t, err)
	require.Equal(t, 1, n)

	n, err = repo.CountVotesBySession("s1", "missing")
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

// TestVotingRepository_BootstrapsVoteSessions proves the self-bootstrap path
// adds the session mapping table without altering the legacy votes table.
func TestVotingRepository_BootstrapsVoteSessions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE votes (
			id TEXT PRIMARY KEY,
			voter_pk TEXT NOT NULL,
			candidate_id TEXT NOT NULL,
			signature TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			block_height INTEGER NOT NULL
		);
		INSERT INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		VALUES
			('legacy', 'voter', 'cand', 'sig', 'msg', 1, 1),
			('mapped', 'voter', 'cand', 'sig', 'msg', 2, 2);
		CREATE TABLE vote_sessions (
			vote_id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL
		);
		INSERT INTO vote_sessions (vote_id, session_id) VALUES ('mapped', 's-existing');
		CREATE TABLE voters (
			public_key TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			has_voted INTEGER NOT NULL DEFAULT 0,
			vote_hash TEXT,
			registered_at INTEGER NOT NULL
		);
		INSERT INTO voters (public_key, name, has_voted, vote_hash, registered_at)
		VALUES ('voter', 'Legacy voter', 0, NULL, 1);
	`)
	require.NoError(t, err)

	repo, err := NewVotingRepository(db)
	require.NoError(t, err)

	var tableCount int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'vote_sessions'`,
	).Scan(&tableCount))
	require.Equal(t, 1, tableCount)

	// A vote written before the session column existed must read back with an
	// empty session id, not an error or a fabricated one. The repository's
	// GetVote accessor was removed as dead code, so assert the stored value
	// directly.
	var legacySession string
	require.NoError(t, db.QueryRow(
		`SELECT COALESCE(vs.session_id, '') FROM votes v
		 LEFT JOIN vote_sessions vs ON vs.vote_id = v.id WHERE v.id = 'legacy'`,
	).Scan(&legacySession))
	require.Equal(t, "", legacySession, "legacy votes cannot be attributed to a session")

	var claimVoteID string
	require.NoError(t, db.QueryRow(
		`SELECT vote_id FROM session_vote_claims
		 WHERE session_id = 's-existing' AND voter_pk = 'voter'`,
	).Scan(&claimVoteID))
	require.Equal(t, "mapped", claimVoteID,
		"self-bootstrap must backfill claims from existing session mappings")
	legacyVoter, err := repo.GetVoter("voter")
	require.NoError(t, err)
	require.True(t, legacyVoter.HasVoted)

	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "v2", Name: "Voter", RegisteredAt: 1}))
	require.NoError(t, repo.SaveVote(voting.NewSessionVote("v2", "cand", "s1", "sig", "msg")))
	n, err := repo.CountVotesBySession("s1", "cand")
	require.NoError(t, err)
	require.Equal(t, 1, n)
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

	voters, err := repo.ListVoters(0, 0)
	require.NoError(t, err)
	require.Len(t, voters, 3)
	require.Equal(t, "Voter 2", voters[0].Name, "should be sorted by registered_at DESC")
	require.Equal(t, "Voter 3", voters[1].Name)
	require.Equal(t, "Voter 1", voters[2].Name)
}

func TestVotingRepository_ListVoters_Empty(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	voters, err := repo.ListVoters(0, 0)
	require.NoError(t, err)
	require.Empty(t, voters)
}

func TestVotingRepository_ListVotersPages(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	// Same-second registrations must keep a stable order across requests.
	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-a", Name: "A", RegisteredAt: 200}))
	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-c", Name: "C", RegisteredAt: 200}))
	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-b", Name: "B", RegisteredAt: 100}))
	// Exercise the SQL tie-breaker itself, not an incidental index order.
	_, err := repo.db.Exec(`DROP INDEX idx_voters_page`)
	require.NoError(t, err)
	_, err = repo.db.Exec(`CREATE INDEX idx_voters_registered_legacy ON voters(registered_at)`)
	require.NoError(t, err)

	first, err := repo.ListVoters(1, 0)
	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, "pk-a", first[0].PublicKey, "same-second rows use public_key as the tie-breaker")

	page, err := repo.ListVoters(1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "pk-c", page[0].PublicKey)

	past, err := repo.ListVoters(10, 99)
	require.NoError(t, err)
	require.Empty(t, past)
}

func TestVotingRepository_Counts(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	counts, err := repo.Counts()
	require.NoError(t, err)
	require.Equal(t, voting.Counts{}, counts, "empty tables must report zero, not a NULL sum")

	require.NoError(t, repo.SaveCandidate(&voting.Candidate{ID: "c1", Name: "C1", VoteCount: 7, CreatedAt: 1}))
	require.NoError(t, repo.SaveCandidate(&voting.Candidate{ID: "c2", Name: "C2", VoteCount: 5, CreatedAt: 2}))
	require.NoError(t, repo.SaveCandidate(&voting.Candidate{ID: "c3", Name: "C3", VoteCount: 0, CreatedAt: 3}))
	require.NoError(t, repo.SaveSession(&voting.Session{ID: "s1", Title: "S1", CreatedAt: 1}))
	require.NoError(t, repo.SaveSession(&voting.Session{ID: "s2", Title: "S2", CreatedAt: 2}))

	counts, err = repo.Counts()
	require.NoError(t, err)
	require.Equal(t, voting.Counts{Candidates: 3, Votes: 12, Sessions: 2}, counts)
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

	sessions, err := repo.ListSessions(0, 0)
	require.NoError(t, err)
	require.Len(t, sessions, 3)
	require.Equal(t, "Session 2", sessions[0].Title, "should be sorted by created_at DESC")
	require.Equal(t, "Session 3", sessions[1].Title)
	require.Equal(t, "Session 1", sessions[2].Title)
}

func TestVotingRepository_ListSessionsPages(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	for i, title := range []string{"First", "Second", "Third"} {
		session := voting.NewSession(title, "Desc", []string{"c1"}, 1000, 2000)
		session.CreatedAt = int64(100 + i*10)
		require.NoError(t, repo.SaveSession(session))
	}

	page, err := repo.ListSessions(1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "Second", page[0].Title, "the second row in newest-first order")

	past, err := repo.ListSessions(10, 99)
	require.NoError(t, err)
	require.Empty(t, past, "an offset past the end is an empty page")
}

func TestVotingRepository_ListSessions_Empty(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	sessions, err := repo.ListSessions(0, 0)
	require.NoError(t, err)
	require.Empty(t, sessions)
}

// =================================================================
// WithTx — transaction-scoped repository
// =================================================================

// TestVotingRepository_WithTx_CommitAndVisibility verifies that writes
// through a tx-scoped repository are visible to reads in the same
// transaction before commit, and durable after commit.
func TestVotingRepository_WithTx_CommitAndVisibility(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-tx", Name: "Tx Voter", RegisteredAt: 1}))
	require.NoError(t, repo.SaveCandidate(&voting.Candidate{ID: "cand-tx", Name: "Tx Cand"}))

	txMgr := NewTxManager(repo.db)
	err := txMgr.WithTransaction(func(tx *sql.Tx) error {
		txRepo := repo.WithTx(tx)

		vote := voting.NewSessionVote("pk-tx", "cand-tx", "session-tx", "sig", "msg")
		if err := txRepo.SaveVote(vote); err != nil {
			return err
		}
		if err := txRepo.IncrementCandidateVoteCount("cand-tx"); err != nil {
			return err
		}

		// Same-tx visibility: the claim and the tally increment must be
		// readable through the tx-scoped repo before commit.
		v, err := txRepo.GetVoter("pk-tx")
		if err != nil {
			return err
		}
		require.True(t, v.HasVoted, "tx-scoped read must see the uncommitted claim")

		c, err := txRepo.GetCandidate("cand-tx")
		if err != nil {
			return err
		}
		require.Equal(t, 1, c.VoteCount, "tx-scoped read must see the uncommitted tally")
		return nil
	})
	require.NoError(t, err)

	// After commit the state is visible through the plain pool repo.
	v, err := repo.GetVoter("pk-tx")
	require.NoError(t, err)
	require.True(t, v.HasVoted)
	c, err := repo.GetCandidate("cand-tx")
	require.NoError(t, err)
	require.Equal(t, 1, c.VoteCount)
	votes, err := repo.GetVotesByVoter("pk-tx")
	require.NoError(t, err)
	require.Len(t, votes, 1)
}

// TestVotingRepository_WithTx_RollbackLeavesNoPartialState proves that when
// the ballot transaction fails after the voter claim and the vote row, ALL
// writes roll back — the voter is not locked out, no orphan vote row
// survives, and the tally is untouched. This replaces the old best-effort
// UnmarkVoted/DeleteVote compensation.
func TestVotingRepository_WithTx_RollbackLeavesNoPartialState(t *testing.T) {
	repo, cleanup := setupVotingTestDB(t)
	defer cleanup()

	require.NoError(t, repo.SaveVoter(&voting.Voter{PublicKey: "pk-rb", Name: "Rollback Voter", RegisteredAt: 1}))
	require.NoError(t, repo.SaveCandidate(&voting.Candidate{ID: "cand-rb", Name: "Rollback Cand", VoteCount: 5}))

	boom := errors.New("mid-transaction failure")
	txMgr := NewTxManager(repo.db)
	err := txMgr.WithTransaction(func(tx *sql.Tx) error {
		txRepo := repo.WithTx(tx)

		vote := voting.NewSessionVote("pk-rb", "cand-rb", "session-rb", "sig", "msg")
		if err := txRepo.SaveVote(vote); err != nil {
			return err
		}
		return boom // force rollback after claim + vote row
	})
	require.ErrorIs(t, err, boom)

	v, err := repo.GetVoter("pk-rb")
	require.NoError(t, err)
	require.False(t, v.HasVoted, "voter claim must be rolled back")
	require.Empty(t, v.VoteHash, "vote hash must be rolled back")

	votes, err := repo.GetVotesByVoter("pk-rb")
	require.NoError(t, err)
	require.Empty(t, votes, "vote row must be rolled back")

	c, err := repo.GetCandidate("cand-rb")
	require.NoError(t, err)
	require.Equal(t, 5, c.VoteCount, "tally must be untouched")
}
