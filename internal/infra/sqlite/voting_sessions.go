// The session registry: governance windows, their roster and lifecycle, plus
// the one cross-entity count the dashboard reads.
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/voting"
)

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
