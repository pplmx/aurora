// The candidate registry: the standing contenders a session's roster selects
// from, and the denormalised tally the ballot path keeps in step with the votes.
package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/voting"
)

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
