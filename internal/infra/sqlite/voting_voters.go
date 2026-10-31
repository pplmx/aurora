// The voter registry: one signing key per voter, plus the lifetime "has voted"
// mark a successful ballot claim sets.
package sqlite

import (
	"database/sql"

	"github.com/pplmx/aurora/internal/domain/voting"
)

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
