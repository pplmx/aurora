package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/voting"
)

type VotingRepository struct {
	db *sql.DB
}

func NewVotingRepository(db *sql.DB) *VotingRepository {
	return &VotingRepository{db: db}
}

func (r *VotingRepository) SaveVote(vote *voting.Vote) error {
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		vote.ID, vote.VoterPublicKey, vote.CandidateID, vote.Signature, vote.Message, vote.Timestamp, vote.BlockHeight,
	)
	return err
}

func (r *VotingRepository) GetVote(id string) (*voting.Vote, error) {
	row := r.db.QueryRow(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE id = ?`,
		id,
	)
	v := &voting.Vote{}
	err := row.Scan(&v.ID, &v.VoterPublicKey, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

func (r *VotingRepository) GetVotesByCandidate(candidateID string) ([]*voting.Vote, error) {
	rows, err := r.db.Query(
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
	rows, err := r.db.Query(
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

func (r *VotingRepository) SaveVoter(voter *voting.Voter) error {
	hasVoted := 0
	if voter.HasVoted {
		hasVoted = 1
	}
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO voters (public_key, name, has_voted, vote_hash, registered_at)
		 VALUES (?, ?, ?, ?, ?)`,
		voter.PublicKey, voter.Name, hasVoted, voter.VoteHash, voter.RegisteredAt,
	)
	return err
}

func (r *VotingRepository) GetVoter(pk string) (*voting.Voter, error) {
	row := r.db.QueryRow(
		`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters WHERE public_key = ?`,
		pk,
	)
	v := &voting.Voter{}
	var hasVoted int
	err := row.Scan(&v.PublicKey, &v.Name, &hasVoted, &v.VoteHash, &v.RegisteredAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	v.HasVoted = hasVoted == 1
	return v, err
}

func (r *VotingRepository) UpdateVoter(voter *voting.Voter) error {
	hasVoted := 0
	if voter.HasVoted {
		hasVoted = 1
	}
	_, err := r.db.Exec(
		`UPDATE voters SET name = ?, has_voted = ?, vote_hash = ? WHERE public_key = ?`,
		voter.Name, hasVoted, voter.VoteHash, voter.PublicKey,
	)
	return err
}

func (r *VotingRepository) ListVoters() ([]*voting.Voter, error) {
	rows, err := r.db.Query(
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
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO candidates (id, name, party, program, description, image_url, vote_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		candidate.ID, candidate.Name, candidate.Party, candidate.Program, "", candidate.ImageURL, candidate.VoteCount, candidate.CreatedAt,
	)
	return err
}

func (r *VotingRepository) GetCandidate(id string) (*voting.Candidate, error) {
	row := r.db.QueryRow(
		`SELECT id, name, party, program, description, image_url, vote_count, created_at FROM candidates WHERE id = ?`,
		id,
	)
	c := &voting.Candidate{}
	var desc string
	err := row.Scan(&c.ID, &c.Name, &c.Party, &c.Program, &desc, &c.ImageURL, &c.VoteCount, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (r *VotingRepository) UpdateCandidate(candidate *voting.Candidate) error {
	_, err := r.db.Exec(
		`UPDATE candidates SET name = ?, party = ?, program = ?, image_url = ?, vote_count = ? WHERE id = ?`,
		candidate.Name, candidate.Party, candidate.Program, candidate.ImageURL, candidate.VoteCount, candidate.ID,
	)
	return err
}

func (r *VotingRepository) DeleteCandidate(id string) error {
	_, err := r.db.Exec(`DELETE FROM candidates WHERE id = ?`, id)
	return err
}

func (r *VotingRepository) ListCandidates() ([]*voting.Candidate, error) {
	rows, err := r.db.Query(
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
	_, err = r.db.Exec(
		`INSERT OR REPLACE INTO voting_sessions (id, title, description, start_time, end_time, status, candidates, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Title, session.Description, session.StartTime, session.EndTime, session.Status, string(candidatesJSON), session.CreatedAt,
	)
	return err
}

func (r *VotingRepository) GetSession(id string) (*voting.Session, error) {
	row := r.db.QueryRow(
		`SELECT id, title, description, start_time, end_time, status, candidates, created_at FROM voting_sessions WHERE id = ?`,
		id,
	)
	session := &voting.Session{}
	var candidatesJSON string
	err := row.Scan(&session.ID, &session.Title, &session.Description, &session.StartTime, &session.EndTime, &session.Status, &candidatesJSON, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
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
	_, err = r.db.Exec(
		`UPDATE voting_sessions SET title = ?, description = ?, start_time = ?, end_time = ?, status = ?, candidates = ? WHERE id = ?`,
		session.Title, session.Description, session.StartTime, session.EndTime, session.Status, string(candidatesJSON), session.ID,
	)
	return err
}

func (r *VotingRepository) ListSessions() ([]*voting.Session, error) {
	rows, err := r.db.Query(
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
