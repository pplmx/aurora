// The voting repository: its schema, the transaction plumbing every method
// shares, and the vote records themselves. The three registries the voting flow
// reads and writes live beside it so each can be read as one question — voters
// in voting_voters.go, candidates in voting_candidates.go, sessions in
// voting_sessions.go — instead of scrolling a single table-by-table file.
package sqlite

import (
	"database/sql"
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
		`INSERT INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
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
			`INSERT INTO vote_sessions (vote_id, session_id) VALUES (?, ?)`,
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
		 ON CONFLICT(session_id, voter_pk) DO NOTHING`,
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

func (r *VotingRepository) GetVotesByCandidate(candidateID string) ([]*voting.Vote, error) {
	rows, err := r.q().Query(
		`SELECT v.id, v.voter_pk, v.candidate_id, COALESCE(vs.session_id, ''),
		        v.signature, v.message, v.timestamp, v.block_height
		 FROM votes v
		 LEFT JOIN vote_sessions vs ON vs.vote_id = v.id
		 WHERE v.candidate_id = ?
		 ORDER BY v.timestamp DESC, v.id DESC`,
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
		 ORDER BY v.timestamp DESC, v.id DESC`,
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
