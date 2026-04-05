package voting

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type DBCandidate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Party       string `json:"party"`
	Program     string `json:"program"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	VoteCount   int    `json:"vote_count"`
	CreatedAt   int64  `json:"created_at"`
}

type DBVoter struct {
	PublicKey    string `json:"public_key"`
	Name         string `json:"name"`
	HasVoted     bool   `json:"has_voted"`
	VoteHash     string `json:"vote_hash"`
	RegisteredAt int64  `json:"registered_at"`
}

type DBVoteRecord struct {
	ID          string `json:"id"`
	VoterPK     string `json:"voter_pk"`
	CandidateID string `json:"candidate_id"`
	Signature   string `json:"signature"`
	Message     string `json:"message"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type DBVotingSession struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	StartTime   int64    `json:"start_time"`
	EndTime     int64    `json:"end_time"`
	Status      string   `json:"status"`
	Candidates  []string `json:"candidates"`
	CreatedAt   int64    `json:"created_at"`
}

type Storage interface {
	SaveCandidate(c *DBCandidate) error
	GetCandidate(id string) (*DBCandidate, error)
	ListCandidates() ([]*DBCandidate, error)
	UpdateCandidate(c *DBCandidate) error
	DeleteCandidate(id string) error

	SaveVoter(v *DBVoter) error
	GetVoter(pk string) (*DBVoter, error)
	UpdateVoter(v *DBVoter) error
	ListVoters() ([]*DBVoter, error)

	SaveVote(v *DBVoteRecord) error
	GetVote(id string) (*DBVoteRecord, error)
	GetVotesByCandidate(candidateID string) ([]*DBVoteRecord, error)
	GetVotesByVoter(voterPK string) ([]*DBVoteRecord, error)
	ListVotes() ([]*DBVoteRecord, error)

	SaveSession(s *DBVotingSession) error
	GetSession(id string) (*DBVotingSession, error)
	ListSessions() ([]*DBVotingSession, error)
	UpdateSession(s *DBVotingSession) error

	Begin() error
	Commit() error
	Rollback() error
	Close() error
}

type SQLiteStorage struct {
	db *sql.DB
	tx *sql.Tx
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &SQLiteStorage{db: db}
	if err := s.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}
	return s, nil
}

func NewSQLiteStorageWithDB(db *sql.DB) *SQLiteStorage {
	s := &SQLiteStorage{db: db}
	s.initTables()
	return s
}

func (s *SQLiteStorage) initTables() error {
	if _, err := s.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS candidates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			party TEXT NOT NULL,
			program TEXT,
			description TEXT,
			image_url TEXT,
			vote_count INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS voters (
			public_key TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			has_voted INTEGER DEFAULT 0,
			vote_hash TEXT,
			registered_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS votes (
			id TEXT PRIMARY KEY,
			voter_pk TEXT NOT NULL,
			candidate_id TEXT NOT NULL,
			signature TEXT NOT NULL,
			message TEXT,
			timestamp INTEGER NOT NULL,
			block_height INTEGER DEFAULT 0,
			FOREIGN KEY (voter_pk) REFERENCES voters(public_key),
			FOREIGN KEY (candidate_id) REFERENCES candidates(id)
		)`,
		`CREATE TABLE IF NOT EXISTS voting_sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			start_time INTEGER NOT NULL,
			end_time INTEGER NOT NULL,
			status TEXT NOT NULL,
			candidates TEXT,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_candidates_name ON candidates(name)`,
		`CREATE INDEX IF NOT EXISTS idx_voters_public_key ON voters(public_key)`,
		`CREATE INDEX IF NOT EXISTS idx_votes_voter_pk ON votes(voter_pk)`,
		`CREATE INDEX IF NOT EXISTS idx_votes_candidate_id ON votes(candidate_id)`,
		`CREATE INDEX IF NOT EXISTS idx_voting_sessions_status ON voting_sessions(status)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to exec query: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStorage) SaveCandidate(c *DBCandidate) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO candidates (id, name, party, program, description, image_url, vote_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.Party, c.Program, c.Description, c.ImageURL, c.VoteCount, c.CreatedAt,
	)
	return err
}

func (s *SQLiteStorage) GetCandidate(id string) (*DBCandidate, error) {
	row := s.db.QueryRow(
		`SELECT id, name, party, program, description, image_url, vote_count, created_at FROM candidates WHERE id = ?`,
		id,
	)
	c := &DBCandidate{}
	err := row.Scan(&c.ID, &c.Name, &c.Party, &c.Program, &c.Description, &c.ImageURL, &c.VoteCount, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (s *SQLiteStorage) ListCandidates() ([]*DBCandidate, error) {
	rows, err := s.db.Query(
		`SELECT id, name, party, program, description, image_url, vote_count, created_at FROM candidates ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []*DBCandidate
	for rows.Next() {
		c := &DBCandidate{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Party, &c.Program, &c.Description, &c.ImageURL, &c.VoteCount, &c.CreatedAt); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

func (s *SQLiteStorage) UpdateCandidate(c *DBCandidate) error {
	_, err := s.db.Exec(
		`UPDATE candidates SET name = ?, party = ?, program = ?, description = ?, image_url = ?, vote_count = ? WHERE id = ?`,
		c.Name, c.Party, c.Program, c.Description, c.ImageURL, c.VoteCount, c.ID,
	)
	return err
}

func (s *SQLiteStorage) DeleteCandidate(id string) error {
	_, err := s.db.Exec(`DELETE FROM candidates WHERE id = ?`, id)
	return err
}

func (s *SQLiteStorage) SaveVoter(v *DBVoter) error {
	hasVoted := 0
	if v.HasVoted {
		hasVoted = 1
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO voters (public_key, name, has_voted, vote_hash, registered_at)
		 VALUES (?, ?, ?, ?, ?)`,
		v.PublicKey, v.Name, hasVoted, v.VoteHash, v.RegisteredAt,
	)
	return err
}

func (s *SQLiteStorage) GetVoter(pk string) (*DBVoter, error) {
	row := s.db.QueryRow(
		`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters WHERE public_key = ?`,
		pk,
	)
	v := &DBVoter{}
	var hasVoted int
	err := row.Scan(&v.PublicKey, &v.Name, &hasVoted, &v.VoteHash, &v.RegisteredAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	v.HasVoted = hasVoted == 1
	return v, err
}

func (s *SQLiteStorage) UpdateVoter(v *DBVoter) error {
	hasVoted := 0
	if v.HasVoted {
		hasVoted = 1
	}
	_, err := s.db.Exec(
		`UPDATE voters SET name = ?, has_voted = ?, vote_hash = ? WHERE public_key = ?`,
		v.Name, hasVoted, v.VoteHash, v.PublicKey,
	)
	return err
}

func (s *SQLiteStorage) ListVoters() ([]*DBVoter, error) {
	rows, err := s.db.Query(
		`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters ORDER BY registered_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voters []*DBVoter
	for rows.Next() {
		v := &DBVoter{}
		var hasVoted int
		if err := rows.Scan(&v.PublicKey, &v.Name, &hasVoted, &v.VoteHash, &v.RegisteredAt); err != nil {
			return nil, err
		}
		v.HasVoted = hasVoted == 1
		voters = append(voters, v)
	}
	return voters, rows.Err()
}

func (s *SQLiteStorage) SaveVote(v *DBVoteRecord) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.VoterPK, v.CandidateID, v.Signature, v.Message, v.Timestamp, v.BlockHeight,
	)
	return err
}

func (s *SQLiteStorage) GetVote(id string) (*DBVoteRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE id = ?`,
		id,
	)
	v := &DBVoteRecord{}
	err := row.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

func (s *SQLiteStorage) GetVotesByCandidate(candidateID string) ([]*DBVoteRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE candidate_id = ? ORDER BY timestamp DESC`,
		candidateID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*DBVoteRecord
	for rows.Next() {
		v := &DBVoteRecord{}
		if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

func (s *SQLiteStorage) GetVotesByVoter(voterPK string) ([]*DBVoteRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes WHERE voter_pk = ? ORDER BY timestamp DESC`,
		voterPK,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*DBVoteRecord
	for rows.Next() {
		v := &DBVoteRecord{}
		if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

func (s *SQLiteStorage) ListVotes() ([]*DBVoteRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes ORDER BY timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*DBVoteRecord
	for rows.Next() {
		v := &DBVoteRecord{}
		if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}
	return votes, rows.Err()
}

func (s *SQLiteStorage) SaveSession(session *DBVotingSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	candidatesJSON, _ := json.Marshal(session.Candidates)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO voting_sessions (id, title, description, start_time, end_time, status, candidates, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Title, session.Description, session.StartTime, session.EndTime, session.Status, string(candidatesJSON), session.CreatedAt,
	)
	return err
}

func (s *SQLiteStorage) GetSession(id string) (*DBVotingSession, error) {
	row := s.db.QueryRow(
		`SELECT id, title, description, start_time, end_time, status, candidates, created_at FROM voting_sessions WHERE id = ?`,
		id,
	)
	session := &DBVotingSession{}
	var candidatesJSON string
	err := row.Scan(&session.ID, &session.Title, &session.Description, &session.StartTime, &session.EndTime, &session.Status, &candidatesJSON, &session.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	json.Unmarshal([]byte(candidatesJSON), &session.Candidates)
	return session, err
}

func (s *SQLiteStorage) ListSessions() ([]*DBVotingSession, error) {
	rows, err := s.db.Query(
		`SELECT id, title, description, start_time, end_time, status, candidates, created_at FROM voting_sessions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*DBVotingSession
	for rows.Next() {
		session := &DBVotingSession{}
		var candidatesJSON string
		if err := rows.Scan(&session.ID, &session.Title, &session.Description, &session.StartTime, &session.EndTime, &session.Status, &candidatesJSON, &session.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(candidatesJSON), &session.Candidates)
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (s *SQLiteStorage) UpdateSession(session *DBVotingSession) error {
	candidatesJSON, _ := json.Marshal(session.Candidates)
	_, err := s.db.Exec(
		`UPDATE voting_sessions SET title = ?, description = ?, start_time = ?, end_time = ?, status = ?, candidates = ? WHERE id = ?`,
		session.Title, session.Description, session.StartTime, session.EndTime, session.Status, string(candidatesJSON), session.ID,
	)
	return err
}

func (s *SQLiteStorage) Begin() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx
	return nil
}

func (s *SQLiteStorage) Commit() error {
	if s.tx == nil {
		return nil
	}
	err := s.tx.Commit()
	s.tx = nil
	return err
}

func (s *SQLiteStorage) Rollback() error {
	if s.tx == nil {
		return nil
	}
	err := s.tx.Rollback()
	s.tx = nil
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

type InMemoryStorage struct {
	candidates map[string]*DBCandidate
	voters     map[string]*DBVoter
	votes      map[string]*DBVoteRecord
	sessions   map[string]*DBVotingSession
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		candidates: make(map[string]*DBCandidate),
		voters:     make(map[string]*DBVoter),
		votes:      make(map[string]*DBVoteRecord),
		sessions:   make(map[string]*DBVotingSession),
	}
}

func (s *InMemoryStorage) SaveCandidate(c *DBCandidate) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	s.candidates[c.ID] = c
	return nil
}

func (s *InMemoryStorage) GetCandidate(id string) (*DBCandidate, error) {
	return s.candidates[id], nil
}

func (s *InMemoryStorage) ListCandidates() ([]*DBCandidate, error) {
	var candidates []*DBCandidate
	for _, c := range s.candidates {
		candidates = append(candidates, c)
	}
	return candidates, nil
}

func (s *InMemoryStorage) UpdateCandidate(c *DBCandidate) error {
	s.candidates[c.ID] = c
	return nil
}

func (s *InMemoryStorage) DeleteCandidate(id string) error {
	delete(s.candidates, id)
	return nil
}

func (s *InMemoryStorage) SaveVoter(v *DBVoter) error {
	s.voters[v.PublicKey] = v
	return nil
}

func (s *InMemoryStorage) GetVoter(pk string) (*DBVoter, error) {
	return s.voters[pk], nil
}

func (s *InMemoryStorage) UpdateVoter(v *DBVoter) error {
	s.voters[v.PublicKey] = v
	return nil
}

func (s *InMemoryStorage) ListVoters() ([]*DBVoter, error) {
	var voters []*DBVoter
	for _, v := range s.voters {
		voters = append(voters, v)
	}
	return voters, nil
}

func (s *InMemoryStorage) SaveVote(v *DBVoteRecord) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	s.votes[v.ID] = v
	return nil
}

func (s *InMemoryStorage) GetVote(id string) (*DBVoteRecord, error) {
	return s.votes[id], nil
}

func (s *InMemoryStorage) GetVotesByCandidate(candidateID string) ([]*DBVoteRecord, error) {
	var votes []*DBVoteRecord
	for _, v := range s.votes {
		if v.CandidateID == candidateID {
			votes = append(votes, v)
		}
	}
	return votes, nil
}

func (s *InMemoryStorage) GetVotesByVoter(voterPK string) ([]*DBVoteRecord, error) {
	var votes []*DBVoteRecord
	for _, v := range s.votes {
		if v.VoterPK == voterPK {
			votes = append(votes, v)
		}
	}
	return votes, nil
}

func (s *InMemoryStorage) ListVotes() ([]*DBVoteRecord, error) {
	var votes []*DBVoteRecord
	for _, v := range s.votes {
		votes = append(votes, v)
	}
	return votes, nil
}

func (s *InMemoryStorage) SaveSession(session *DBVotingSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemoryStorage) GetSession(id string) (*DBVotingSession, error) {
	return s.sessions[id], nil
}

func (s *InMemoryStorage) ListSessions() ([]*DBVotingSession, error) {
	var sessions []*DBVotingSession
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *InMemoryStorage) UpdateSession(session *DBVotingSession) error {
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemoryStorage) Begin() error {
	return nil
}

func (s *InMemoryStorage) Commit() error {
	return nil
}

func (s *InMemoryStorage) Rollback() error {
	return nil
}

func (s *InMemoryStorage) Close() error {
	return nil
}
