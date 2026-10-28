package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/voting"
	_ "modernc.org/sqlite"
)

// ErrAlreadyVoted is an alias for the domain sentinel. The repository returns
// it so the use case can classify a lost race without importing this package.
var ErrAlreadyVoted = voting.ErrAlreadyVoted

// votingExec abstracts *sql.DB and *sql.Tx so VotingRepository can run
// either against the pool or inside an explicit transaction.
type votingExec interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type VotingRepository struct {
	db   *sql.DB
	exec votingExec // nil => use db
}

func NewVotingRepository(db *sql.DB) (*VotingRepository, error) {
	repo := &VotingRepository{db: db}
	if err := repo.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create voting tables: %w", err)
	}
	return repo, nil
}

// createTables ensures the voting tables exist on the shared DB handle. Every
// other module (lottery/token/nft/oracle) self-creates its schema, and the
// migrations express the same tables with IF NOT EXISTS, so running this on an
// already-migrated DB is a no-op. Without it a fresh install that has not
// separately run `aurora migrate up` starts in a state where every lottery/
// token/nft/oracle operation works but every voting operation fails with
// "no such table", silently depending on the migrations never running on the
// API path (cmd/api never runs them; migrate.autoRun defaults to false).
func (r *VotingRepository) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS votes (
			id TEXT PRIMARY KEY,
			voter_pk TEXT NOT NULL,
			candidate_id TEXT NOT NULL,
			signature TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			block_height INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_votes_voter ON votes(voter_pk)`,
		`CREATE INDEX IF NOT EXISTS idx_votes_candidate ON votes(candidate_id)`,
		`CREATE TABLE IF NOT EXISTS vote_sessions (
			vote_id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vote_sessions_session_vote ON vote_sessions(session_id, vote_id)`,
		`CREATE TABLE IF NOT EXISTS voters (
			public_key TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			has_voted INTEGER NOT NULL DEFAULT 0,
			vote_hash TEXT,
			registered_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS candidates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			party TEXT NOT NULL,
			program TEXT NOT NULL,
			description TEXT,
			image_url TEXT,
			vote_count INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_candidates_created ON candidates(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_candidates_page ON candidates(created_at DESC, id)`,
		`CREATE TABLE IF NOT EXISTS voting_sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			start_time INTEGER NOT NULL,
			end_time INTEGER NOT NULL,
			status TEXT NOT NULL,
			candidates TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON voting_sessions(status)`,
		// The list reads page newest first. The compound indexes also provide a
		// deterministic id tie-breaker for rows created in the same second.
		// Without an index SQLite scans and sorts every row in a temp B-tree
		// before the caller sees it (measured).
		`CREATE INDEX IF NOT EXISTS idx_sessions_created ON voting_sessions(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_page ON voting_sessions(created_at DESC, id)`,
		// The voter list pages newest first. Keeping public_key in the index
		// gives same-second registrations a stable, unique tie-breaker and
		// replaces the single-column index that could not page deterministically.
		`DROP INDEX IF EXISTS idx_voters_registered`,
		`CREATE INDEX IF NOT EXISTS idx_voters_page ON voters(registered_at DESC, public_key)`,
		`CREATE TABLE IF NOT EXISTS session_vote_claims (
			session_id TEXT NOT NULL,
			voter_pk TEXT NOT NULL,
			vote_id TEXT NOT NULL UNIQUE,
			claimed_at INTEGER NOT NULL,
			PRIMARY KEY (session_id, voter_pk)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_session_vote_claims_voter
			ON session_vote_claims(voter_pk)`,
		// Older binaries can leave session ballots in a self-bootstrapped
		// database before migration 000008 runs. Backfill here so both schema
		// paths enforce the same per-session rule.
		`INSERT OR IGNORE INTO session_vote_claims
			(session_id, voter_pk, vote_id, claimed_at)
		 SELECT vs.session_id, v.voter_pk, v.id, v.timestamp
		 FROM votes v
		 JOIN vote_sessions vs ON vs.vote_id = v.id
		 ORDER BY v.timestamp, v.id`,
		`UPDATE voters
		 SET has_voted = 1
		 WHERE has_voted = 0
		   AND EXISTS (
		       SELECT 1 FROM session_vote_claims c
		       WHERE c.voter_pk = voters.public_key
		   )`,
	}

	for _, query := range queries {
		if _, err := r.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

// q returns the executor this repository reads/writes through: the pooled DB
// by default, or the active *sql.Tx when WithTx was used.
func (r *VotingRepository) q() votingExec {
	if r != nil && r.exec != nil {
		return r.exec
	}
	return r.db
}

// WithTx returns a VotingRepository whose operations run inside the given
// transaction. All reads and writes through the returned repository observe
// the transaction's uncommitted state and participate in its commit/rollback.
func (r *VotingRepository) WithTx(tx *sql.Tx) voting.Repository {
	return &VotingRepository{db: r.db, exec: tx}
}

func (r *VotingRepository) SaveVote(vote *voting.Vote) error {
	if r.exec == nil {
		tx, err := r.db.Begin()
		if err != nil {
			return fmt.Errorf("begin vote save: %w", err)
		}
		txRepo := &VotingRepository{db: r.db, exec: tx}
		if err := txRepo.saveVote(vote); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit vote save: %w", err)
		}
		return nil
	}
	return r.saveVote(vote)
}

func (r *VotingRepository) saveVote(vote *voting.Vote) error {
	_, err := r.q().Exec(
		`INSERT OR REPLACE INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		vote.ID, vote.VoterPublicKey, vote.CandidateID, vote.Signature, vote.Message, vote.Timestamp, vote.BlockHeight,
	)
	if err != nil {
		return err
	}

	// Keep the session mapping separate from the legacy votes table so
	// migration 000003 never has to rebuild votes or discard a mapping that
	// the self-bootstrapping repository already wrote.
	if vote.SessionID == "" {
		_, err = r.q().Exec(`DELETE FROM vote_sessions WHERE vote_id = ?`, vote.ID)
	} else {
		_, err = r.q().Exec(
			`INSERT OR REPLACE INTO vote_sessions (vote_id, session_id) VALUES (?, ?)`,
			vote.ID, vote.SessionID,
		)
	}
	if err != nil || vote.SessionID == "" {
		return err
	}
	return r.claimSessionBallot(vote)
}

// claimSessionBallot is the atomic duplicate-ballot guard for session votes.
// SaveVote always runs in a transaction (its own or CastVote's), so a duplicate
// claim rolls back the vote, session mapping and tally together.
func (r *VotingRepository) claimSessionBallot(vote *voting.Vote) error {
	res, err := r.q().Exec(
		`INSERT INTO session_vote_claims (session_id, voter_pk, vote_id, claimed_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT DO NOTHING`,
		vote.SessionID, vote.VoterPublicKey, vote.ID, vote.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("claim session ballot: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("claim session ballot rows: %w", err)
	}
	if affected == 0 {
		var claimedVoteID string
		err := r.q().QueryRow(
			`SELECT vote_id FROM session_vote_claims
			 WHERE session_id = ? AND voter_pk = ?`,
			vote.SessionID, vote.VoterPublicKey,
		).Scan(&claimedVoteID)
		if err == sql.ErrNoRows {
			return fmt.Errorf("claim session ballot: vote id %s is already used", vote.ID)
		}
		if err != nil {
			return fmt.Errorf("read session ballot claim: %w", err)
		}
		if claimedVoteID == vote.ID {
			return nil
		}
		return ErrAlreadyVoted
	}

	res, err = r.q().Exec(
		`UPDATE voters
		 SET has_voted = 1,
		     vote_hash = CASE
		         WHEN has_voted = 0 OR vote_hash IS NULL OR vote_hash = '' THEN ?
		         ELSE vote_hash
		     END
		 WHERE public_key = ?`,
		vote.Message, vote.VoterPublicKey,
	)
	if err != nil {
		return fmt.Errorf("mark voter lifetime ballot: %w", err)
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark voter lifetime ballot rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteVote removes a vote by id. CastVote no longer needs it for rollback
// (the whole ballot runs in one transaction); it is retained for
// administrative / corrective flows.
func (r *VotingRepository) GetVotesByCandidate(candidateID string) ([]*voting.Vote, error) {
	rows, err := r.q().Query(
		`SELECT v.id, v.voter_pk, v.candidate_id, COALESCE(vs.session_id, ''),
		        v.signature, v.message, v.timestamp, v.block_height
		 FROM votes v
		 LEFT JOIN vote_sessions vs ON vs.vote_id = v.id
		 WHERE v.candidate_id = ?
		 ORDER BY v.timestamp DESC`,
		candidateID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var votes []*voting.Vote
	for rows.Next() {
		v := &voting.Vote{}
		if err := rows.Scan(&v.ID, &v.VoterPublicKey, &v.CandidateID, &v.SessionID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

// CountVotesBySession counts the claim-winning ballots for one candidate in
// one session. This is the source of truth for the results API:
// candidates.vote_count is a global lifetime tally and must not leak across
// sessions (ISS-299). Historical duplicate vote rows remain in the audit trail
// but only their earliest claim owner is eligible for the tally.
func (r *VotingRepository) CountVotesBySession(sessionID, candidateID string) (int, error) {
	var count int
	err := r.q().QueryRow(
		`SELECT COUNT(*)
		 FROM session_vote_claims c
		 JOIN votes v ON v.id = c.vote_id
		 WHERE c.session_id = ? AND v.candidate_id = ?`,
		sessionID, candidateID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count votes for session %s candidate %s: %w", sessionID, candidateID, err)
	}
	return count, nil
}

func (r *VotingRepository) GetVotesByVoter(voterPK string) ([]*voting.Vote, error) {
	rows, err := r.q().Query(
		`SELECT v.id, v.voter_pk, v.candidate_id, COALESCE(vs.session_id, ''),
		        v.signature, v.message, v.timestamp, v.block_height
		 FROM votes v
		 LEFT JOIN vote_sessions vs ON vs.vote_id = v.id
		 WHERE v.voter_pk = ?
		 ORDER BY v.timestamp DESC`,
		voterPK,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var votes []*voting.Vote
	for rows.Next() {
		v := &voting.Vote{}
		if err := rows.Scan(&v.ID, &v.VoterPublicKey, &v.CandidateID, &v.SessionID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

// SaveVoter creates or updates a voter's profile. The semantics
// here are deliberately narrower than a plain INSERT OR REPLACE:
//
//   - On a new public_key: insert with the given name, has_voted=0
//     (unless the caller already set has_voted=1, in which case we
//     respect it — this lets a future bulk-import flow register a
//     voter that already cast a vote elsewhere).
//
//   - On a conflict (existing public_key): update only `name` and
//     `registered_at`. We deliberately do NOT touch `has_voted` or
//     `vote_hash` — those are the integrity-bearing columns, and
//     the only writer is the session claim in SaveVote.
//
// Why this matters: the previous INSERT OR REPLACE silently wiped
// has_voted and vote_hash on every conflict. That meant any caller
// passing an existing public_key with has_voted=false could undo a
// recorded vote and re-enable ballot claims via the re-registration
// path. Today the only caller is RegisterVoterUseCase, which
// generates a fresh key per registration so the bug isn't
// reachable through normal use — but it's a footgun for any
// future migration / bulk-import / admin-reset path.
func (r *VotingRepository) SaveVoter(voter *voting.Voter) error {
	hasVoted := 0
	if voter.HasVoted {
		hasVoted = 1
	}
	_, err := r.q().Exec(
		`INSERT INTO voters (public_key, name, has_voted, vote_hash, registered_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(public_key) DO UPDATE SET
		   name = excluded.name,
		   registered_at = excluded.registered_at`,
		voter.PublicKey, voter.Name, hasVoted, voter.VoteHash, voter.RegisteredAt,
	)
	return err
}

func (r *VotingRepository) GetVoter(pk string) (*voting.Voter, error) {
	row := r.q().QueryRow(
		`SELECT public_key, name, has_voted, COALESCE(vote_hash, ''), registered_at
		 FROM voters WHERE public_key = ?`,
		pk,
	)
	v := &voting.Voter{}
	var hasVoted int
	err := row.Scan(&v.PublicKey, &v.Name, &hasVoted, &v.VoteHash, &v.RegisteredAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	v.HasVoted = hasVoted == 1
	return v, err
}

func (r *VotingRepository) UpdateVoter(voter *voting.Voter) error {
	hasVoted := 0
	if voter.HasVoted {
		hasVoted = 1
	}
	_, err := r.q().Exec(
		`UPDATE voters SET name = ?, has_voted = ?, vote_hash = ? WHERE public_key = ?`,
		voter.Name, hasVoted, voter.VoteHash, voter.PublicKey,
	)
	return err
}

func (r *VotingRepository) ListVoters(limit, offset int) ([]*voting.Voter, error) {
	if limit <= 0 {
		limit = -1
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.q().Query(
		`SELECT public_key, name, has_voted, COALESCE(vote_hash, ''), registered_at
		 FROM voters ORDER BY registered_at DESC, public_key LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var voters []*voting.Voter
	for rows.Next() {
		v := &voting.Voter{}
		var hasVoted int
		if err := rows.Scan(&v.PublicKey, &v.Name, &hasVoted, &v.VoteHash, &v.RegisteredAt); err != nil {
			return nil, err
		}
		v.HasVoted = hasVoted == 1
		voters = append(voters, v)
	}
	return voters, rows.Err()
}

func (r *VotingRepository) SaveCandidate(candidate *voting.Candidate) error {
	_, err := r.q().Exec(
		`INSERT OR REPLACE INTO candidates (id, name, party, program, description, image_url, vote_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		candidate.ID, candidate.Name, candidate.Party, candidate.Program, "", candidate.ImageURL, candidate.VoteCount, candidate.CreatedAt,
	)
	return err
}

func (r *VotingRepository) GetCandidate(id string) (*voting.Candidate, error) {
	row := r.q().QueryRow(
		`SELECT id, name, party, program, description, image_url, vote_count, created_at FROM candidates WHERE id = ?`,
		id,
	)
	c := &voting.Candidate{}
	var desc string
	err := row.Scan(&c.ID, &c.Name, &c.Party, &c.Program, &desc, &c.ImageURL, &c.VoteCount, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return c, err
}

// IncrementCandidateVoteCount atomically increments a candidate's stored
// vote_count, closing the TOCTOU window in CastVoteUseCase where the count
// was read into a struct, incremented in memory, then written back — under
// concurrent votes to the same candidate one increment was silently lost
// (both readers saw N, both wrote N+1).
//
// The conditional UPDATE relies on SQLite's per-connection write
// serialization: exactly one writer at a time, so no increments are lost.
// Returns ErrNotFound if the candidate does not exist (e.g. deleted
// concurrently between the use case's existence check and this call).
func (r *VotingRepository) IncrementCandidateVoteCount(candidateID string) error {
	res, err := r.q().Exec(
		`UPDATE candidates SET vote_count = vote_count + 1 WHERE id = ?`,
		candidateID,
	)
	if err != nil {
		return fmt.Errorf("increment candidate vote count: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("increment candidate vote count rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VotingRepository) ListCandidates(limit, offset int) ([]*voting.Candidate, error) {
	if limit <= 0 {
		limit = -1
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.q().Query(
		`SELECT id, name, party, program, description, image_url, vote_count, created_at
		 FROM candidates ORDER BY created_at DESC, id LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var candidates []*voting.Candidate
	for rows.Next() {
		c := &voting.Candidate{}
		var desc string
		if err := rows.Scan(&c.ID, &c.Name, &c.Party, &c.Program, &desc, &c.ImageURL, &c.VoteCount, &c.CreatedAt); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

func (r *VotingRepository) SaveSession(session *voting.Session) error {
	candidatesJSON, err := json.Marshal(session.Candidates)
	if err != nil {
		return fmt.Errorf("failed to marshal candidates: %w", err)
	}
	_, err = r.q().Exec(
		`INSERT OR REPLACE INTO voting_sessions (id, title, description, start_time, end_time, status, candidates, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Title, session.Description, session.StartTime, session.EndTime, session.Status, string(candidatesJSON), session.CreatedAt,
	)
	return err
}

func (r *VotingRepository) GetSession(id string) (*voting.Session, error) {
	row := r.q().QueryRow(
		`SELECT id, title, description, start_time, end_time, status, candidates, created_at FROM voting_sessions WHERE id = ?`,
		id,
	)
	session := &voting.Session{}
	var candidatesJSON string
	err := row.Scan(&session.ID, &session.Title, &session.Description, &session.StartTime, &session.EndTime, &session.Status, &candidatesJSON, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err := json.Unmarshal([]byte(candidatesJSON), &session.Candidates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal candidates: %w", err)
	}
	return session, err
}

func (r *VotingRepository) UpdateSession(session *voting.Session) error {
	candidatesJSON, err := json.Marshal(session.Candidates)
	if err != nil {
		return fmt.Errorf("failed to marshal candidates: %w", err)
	}
	_, err = r.q().Exec(
		`UPDATE voting_sessions SET title = ?, description = ?, start_time = ?, end_time = ?, status = ?, candidates = ? WHERE id = ?`,
		session.Title, session.Description, session.StartTime, session.EndTime, session.Status, string(candidatesJSON), session.ID,
	)
	return err
}

func (r *VotingRepository) ListSessions(limit, offset int) ([]*voting.Session, error) {
	if limit <= 0 {
		limit = -1
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.q().Query(
		`SELECT id, title, description, start_time, end_time, status, candidates, created_at
		 FROM voting_sessions ORDER BY created_at DESC, id LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []*voting.Session
	for rows.Next() {
		session := &voting.Session{}
		var candidatesJSON string
		if err := rows.Scan(&session.ID, &session.Title, &session.Description, &session.StartTime, &session.EndTime, &session.Status, &candidatesJSON, &session.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(candidatesJSON), &session.Candidates); err != nil {
			return nil, fmt.Errorf("failed to unmarshal candidates: %w", err)
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *VotingRepository) Counts() (voting.Counts, error) {
	var counts voting.Counts
	err := r.q().QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM candidates),
			(SELECT COALESCE(SUM(vote_count), 0) FROM candidates),
			(SELECT COUNT(*) FROM voting_sessions)
	`).Scan(&counts.Candidates, &counts.Votes, &counts.Sessions)
	return counts, err
}
