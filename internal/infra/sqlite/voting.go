package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/voting"
)

// ErrAlreadyVoted is returned by TryMarkVoted when the voter has already
// cast a vote. The usecase surfaces this as a 409 Conflict at the HTTP
// boundary. Returning a sentinel error (not just sql.ErrNoRows) lets the
// caller distinguish "lost the race" from "no such voter".
var ErrAlreadyVoted = errors.New("voter has already voted")

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

func NewVotingRepository(db *sql.DB) *VotingRepository {
	return &VotingRepository{db: db}
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
	_, err := r.q().Exec(
		`INSERT OR REPLACE INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		vote.ID, vote.VoterPublicKey, vote.CandidateID, vote.Signature, vote.Message, vote.Timestamp, vote.BlockHeight,
	)
	return err
}

func (r *VotingRepository) GetVote(id string) (*voting.Vote, error) {
	row := r.q().QueryRow(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE id = ?`,
		id,
	)
	v := &voting.Vote{}
	err := row.Scan(&v.ID, &v.VoterPublicKey, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return v, err
}

// DeleteVote removes a vote by id. CastVote no longer needs it for rollback
// (the whole ballot runs in one transaction); it is retained for
// administrative / corrective flows.
func (r *VotingRepository) DeleteVote(id string) error {
	_, err := r.q().Exec(`DELETE FROM votes WHERE id = ?`, id)
	return err
}

func (r *VotingRepository) GetVotesByCandidate(candidateID string) ([]*voting.Vote, error) {
	rows, err := r.q().Query(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE candidate_id = ? ORDER BY timestamp DESC`,
		candidateID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var votes []*voting.Vote
	for rows.Next() {
		v := &voting.Vote{}
		if err := rows.Scan(&v.ID, &v.VoterPublicKey, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

func (r *VotingRepository) GetVotesByVoter(voterPK string) ([]*voting.Vote, error) {
	rows, err := r.q().Query(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE voter_pk = ? ORDER BY timestamp DESC`,
		voterPK,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var votes []*voting.Vote
	for rows.Next() {
		v := &voting.Vote{}
		if err := rows.Scan(&v.ID, &v.VoterPublicKey, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
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
//     the only writer to them is TryMarkVoted (a conditional
//     UPDATE that atomically transitions 0→1).
//
// Why this matters: the previous INSERT OR REPLACE silently wiped
// has_voted and vote_hash on every conflict. That meant any caller
// passing an existing public_key with has_voted=false could undo a
// recorded vote, defeating the Round 14 TryMarkVoted atomic
// guarantee and re-enabling double voting via the re-registration
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
		`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters WHERE public_key = ?`,
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

// TryMarkVoted atomically flips has_voted from 0 to 1 for the given
// voter, returning ErrAlreadyVoted if the voter is already marked.
//
// This is the atomic primitive that closes the TOCTOU window in
// CastVoteUseCase: the previous flow read has_voted, decided whether to
// proceed, and then wrote back has_voted=true. Two concurrent requests
// could both pass the read, both sign+save their vote, and both
// UPDATE — silently allowing double voting.
//
// The conditional UPDATE here uses SQLite's per-connection write
// serialization so exactly one caller sees RowsAffected()==1 and the
// rest see RowsAffected()==0 (mapped to ErrAlreadyVoted). voteHash is
// set in the same UPDATE so the audit trail is consistent.
func (r *VotingRepository) TryMarkVoted(publicKey, voteHash string) error {
	res, err := r.q().Exec(
		`UPDATE voters SET has_voted = 1, vote_hash = ? WHERE public_key = ? AND has_voted = 0`,
		voteHash, publicKey,
	)
	if err != nil {
		return fmt.Errorf("try mark voted: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("try mark voted rows: %w", err)
	}
	if affected == 0 {
		// Either the voter doesn't exist or has_voted is already 1.
		// Distinguish so the caller can return the right HTTP code.
		v, getErr := r.GetVoter(publicKey)
		if getErr != nil {
			return getErr // ErrNotFound if missing
		}
		_ = v
		return ErrAlreadyVoted
	}
	return nil
}

// UnmarkVoted resets has_voted to 0 and clears vote_hash for the given
// voter. Historically the best-effort rollback when SaveVote failed after
// TryMarkVoted succeeded; CastVote now runs in a real transaction, so this
// primitive is retained for administrative / corrective flows only.
func (r *VotingRepository) UnmarkVoted(publicKey string) error {
	_, err := r.q().Exec(
		`UPDATE voters SET has_voted = 0, vote_hash = '' WHERE public_key = ?`,
		publicKey,
	)
	return err
}

func (r *VotingRepository) ListVoters() ([]*voting.Voter, error) {
	rows, err := r.q().Query(
		`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters ORDER BY registered_at DESC`,
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

func (r *VotingRepository) UpdateCandidate(candidate *voting.Candidate) error {
	_, err := r.q().Exec(
		`UPDATE candidates SET name = ?, party = ?, program = ?, image_url = ?, vote_count = ? WHERE id = ?`,
		candidate.Name, candidate.Party, candidate.Program, candidate.ImageURL, candidate.VoteCount, candidate.ID,
	)
	return err
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

func (r *VotingRepository) DeleteCandidate(id string) error {
	_, err := r.q().Exec(`DELETE FROM candidates WHERE id = ?`, id)
	return err
}

func (r *VotingRepository) ListCandidates() ([]*voting.Candidate, error) {
	rows, err := r.q().Query(
		`SELECT id, name, party, program, description, image_url, vote_count, created_at FROM candidates ORDER BY created_at DESC`,
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

func (r *VotingRepository) ListSessions() ([]*voting.Session, error) {
	rows, err := r.q().Query(
		`SELECT id, title, description, start_time, end_time, status, candidates, created_at FROM voting_sessions ORDER BY created_at DESC`,
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
