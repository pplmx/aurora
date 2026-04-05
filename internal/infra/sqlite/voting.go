package sqlite

import (
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/voting"
	oldvoting "github.com/pplmx/aurora/internal/voting"
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
		vote.ID, vote.VoterPK, vote.CandidateID, vote.Signature, vote.Message, vote.Timestamp, vote.BlockHeight,
	)
	return err
}

func (r *VotingRepository) GetVote(id string) (*voting.Vote, error) {
	row := r.db.QueryRow(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE id = ?`,
		id,
	)
	v := &voting.Vote{}
	err := row.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight)
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
	defer rows.Close()

	var votes []*voting.Vote
	for rows.Next() {
		v := &voting.Vote{}
		if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
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
	defer rows.Close()

	var votes []*voting.Vote
	for rows.Next() {
		v := &voting.Vote{}
		if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
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
	defer rows.Close()

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
	defer rows.Close()

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
	candidatesJSON, _ := json.Marshal(session.Candidates)
	_, err := r.db.Exec(
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
	json.Unmarshal([]byte(candidatesJSON), &session.Candidates)
	return session, err
}

func (r *VotingRepository) UpdateSession(session *voting.Session) error {
	candidatesJSON, _ := json.Marshal(session.Candidates)
	_, err := r.db.Exec(
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
	defer rows.Close()

	var sessions []*voting.Session
	for rows.Next() {
		session := &voting.Session{}
		var candidatesJSON string
		if err := rows.Scan(&session.ID, &session.Title, &session.Description, &session.StartTime, &session.EndTime, &session.Status, &candidatesJSON, &session.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(candidatesJSON), &session.Candidates)
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

type VotingStorageAdapter struct {
	storage oldvoting.Storage
}

func NewVotingStorageAdapter(storage oldvoting.Storage) *VotingStorageAdapter {
	return &VotingStorageAdapter{storage: storage}
}

func (a *VotingStorageAdapter) SaveVote(vote *voting.Vote) error {
	dbVote := &oldvoting.DBVoteRecord{
		ID:          vote.ID,
		VoterPK:     vote.VoterPK,
		CandidateID: vote.CandidateID,
		Signature:   vote.Signature,
		Message:     vote.Message,
		Timestamp:   vote.Timestamp,
		BlockHeight: vote.BlockHeight,
	}
	return a.storage.SaveVote(dbVote)
}

func (a *VotingStorageAdapter) GetVote(id string) (*voting.Vote, error) {
	dbVote, err := a.storage.GetVote(id)
	if err != nil || dbVote == nil {
		return nil, err
	}
	return &voting.Vote{
		ID:          dbVote.ID,
		VoterPK:     dbVote.VoterPK,
		CandidateID: dbVote.CandidateID,
		Signature:   dbVote.Signature,
		Message:     dbVote.Message,
		Timestamp:   dbVote.Timestamp,
		BlockHeight: dbVote.BlockHeight,
	}, nil
}

func (a *VotingStorageAdapter) GetVotesByCandidate(candidateID string) ([]*voting.Vote, error) {
	dbVotes, err := a.storage.GetVotesByCandidate(candidateID)
	if err != nil {
		return nil, err
	}
	votes := make([]*voting.Vote, len(dbVotes))
	for i, db := range dbVotes {
		votes[i] = &voting.Vote{
			ID:          db.ID,
			VoterPK:     db.VoterPK,
			CandidateID: db.CandidateID,
			Signature:   db.Signature,
			Message:     db.Message,
			Timestamp:   db.Timestamp,
			BlockHeight: db.BlockHeight,
		}
	}
	return votes, nil
}

func (a *VotingStorageAdapter) GetVotesByVoter(voterPK string) ([]*voting.Vote, error) {
	dbVotes, err := a.storage.GetVotesByVoter(voterPK)
	if err != nil {
		return nil, err
	}
	votes := make([]*voting.Vote, len(dbVotes))
	for i, db := range dbVotes {
		votes[i] = &voting.Vote{
			ID:          db.ID,
			VoterPK:     db.VoterPK,
			CandidateID: db.CandidateID,
			Signature:   db.Signature,
			Message:     db.Message,
			Timestamp:   db.Timestamp,
			BlockHeight: db.BlockHeight,
		}
	}
	return votes, nil
}

func (a *VotingStorageAdapter) SaveVoter(voter *voting.Voter) error {
	dbVoter := &oldvoting.DBVoter{
		PublicKey:    voter.PublicKey,
		Name:         voter.Name,
		HasVoted:     voter.HasVoted,
		VoteHash:     voter.VoteHash,
		RegisteredAt: voter.RegisteredAt,
	}
	return a.storage.SaveVoter(dbVoter)
}

func (a *VotingStorageAdapter) GetVoter(pk string) (*voting.Voter, error) {
	dbVoter, err := a.storage.GetVoter(pk)
	if err != nil || dbVoter == nil {
		return nil, err
	}
	return &voting.Voter{
		PublicKey:    dbVoter.PublicKey,
		Name:         dbVoter.Name,
		HasVoted:     dbVoter.HasVoted,
		VoteHash:     dbVoter.VoteHash,
		RegisteredAt: dbVoter.RegisteredAt,
	}, nil
}

func (a *VotingStorageAdapter) UpdateVoter(voter *voting.Voter) error {
	dbVoter := &oldvoting.DBVoter{
		PublicKey:    voter.PublicKey,
		Name:         voter.Name,
		HasVoted:     voter.HasVoted,
		VoteHash:     voter.VoteHash,
		RegisteredAt: voter.RegisteredAt,
	}
	return a.storage.UpdateVoter(dbVoter)
}

func (a *VotingStorageAdapter) ListVoters() ([]*voting.Voter, error) {
	dbVoters, err := a.storage.ListVoters()
	if err != nil {
		return nil, err
	}
	voters := make([]*voting.Voter, len(dbVoters))
	for i, db := range dbVoters {
		voters[i] = &voting.Voter{
			PublicKey:    db.PublicKey,
			Name:         db.Name,
			HasVoted:     db.HasVoted,
			VoteHash:     db.VoteHash,
			RegisteredAt: db.RegisteredAt,
		}
	}
	return voters, nil
}

func (a *VotingStorageAdapter) SaveCandidate(candidate *voting.Candidate) error {
	dbCandidate := &oldvoting.DBCandidate{
		ID:        candidate.ID,
		Name:      candidate.Name,
		Party:     candidate.Party,
		Program:   candidate.Program,
		ImageURL:  candidate.ImageURL,
		VoteCount: candidate.VoteCount,
		CreatedAt: candidate.CreatedAt,
	}
	return a.storage.SaveCandidate(dbCandidate)
}

func (a *VotingStorageAdapter) GetCandidate(id string) (*voting.Candidate, error) {
	dbCandidate, err := a.storage.GetCandidate(id)
	if err != nil || dbCandidate == nil {
		return nil, err
	}
	return &voting.Candidate{
		ID:        dbCandidate.ID,
		Name:      dbCandidate.Name,
		Party:     dbCandidate.Party,
		Program:   dbCandidate.Program,
		ImageURL:  dbCandidate.ImageURL,
		VoteCount: dbCandidate.VoteCount,
		CreatedAt: dbCandidate.CreatedAt,
	}, nil
}

func (a *VotingStorageAdapter) UpdateCandidate(candidate *voting.Candidate) error {
	dbCandidate := &oldvoting.DBCandidate{
		ID:        candidate.ID,
		Name:      candidate.Name,
		Party:     candidate.Party,
		Program:   candidate.Program,
		ImageURL:  candidate.ImageURL,
		VoteCount: candidate.VoteCount,
		CreatedAt: candidate.CreatedAt,
	}
	return a.storage.UpdateCandidate(dbCandidate)
}

func (a *VotingStorageAdapter) DeleteCandidate(id string) error {
	return a.storage.DeleteCandidate(id)
}

func (a *VotingStorageAdapter) ListCandidates() ([]*voting.Candidate, error) {
	dbCandidates, err := a.storage.ListCandidates()
	if err != nil {
		return nil, err
	}
	candidates := make([]*voting.Candidate, len(dbCandidates))
	for i, db := range dbCandidates {
		candidates[i] = &voting.Candidate{
			ID:        db.ID,
			Name:      db.Name,
			Party:     db.Party,
			Program:   db.Program,
			ImageURL:  db.ImageURL,
			VoteCount: db.VoteCount,
			CreatedAt: db.CreatedAt,
		}
	}
	return candidates, nil
}

func (a *VotingStorageAdapter) SaveSession(session *voting.Session) error {
	dbSession := &oldvoting.DBVotingSession{
		ID:          session.ID,
		Title:       session.Title,
		Description: session.Description,
		StartTime:   session.StartTime,
		EndTime:     session.EndTime,
		Status:      session.Status,
		Candidates:  session.Candidates,
		CreatedAt:   session.CreatedAt,
	}
	return a.storage.SaveSession(dbSession)
}

func (a *VotingStorageAdapter) GetSession(id string) (*voting.Session, error) {
	dbSession, err := a.storage.GetSession(id)
	if err != nil || dbSession == nil {
		return nil, err
	}
	return &voting.Session{
		ID:          dbSession.ID,
		Title:       dbSession.Title,
		Description: dbSession.Description,
		StartTime:   dbSession.StartTime,
		EndTime:     dbSession.EndTime,
		Status:      dbSession.Status,
		Candidates:  dbSession.Candidates,
		CreatedAt:   dbSession.CreatedAt,
	}, nil
}

func (a *VotingStorageAdapter) UpdateSession(session *voting.Session) error {
	dbSession := &oldvoting.DBVotingSession{
		ID:          session.ID,
		Title:       session.Title,
		Description: session.Description,
		StartTime:   session.StartTime,
		EndTime:     session.EndTime,
		Status:      session.Status,
		Candidates:  session.Candidates,
		CreatedAt:   session.CreatedAt,
	}
	return a.storage.UpdateSession(dbSession)
}

func (a *VotingStorageAdapter) ListSessions() ([]*voting.Session, error) {
	dbSessions, err := a.storage.ListSessions()
	if err != nil {
		return nil, err
	}
	sessions := make([]*voting.Session, len(dbSessions))
	for i, db := range dbSessions {
		sessions[i] = &voting.Session{
			ID:          db.ID,
			Title:       db.Title,
			Description: db.Description,
			StartTime:   db.StartTime,
			EndTime:     db.EndTime,
			Status:      db.Status,
			Candidates:  db.Candidates,
			CreatedAt:   db.CreatedAt,
		}
	}
	return sessions, nil
}
