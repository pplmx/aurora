package oracle

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type DataSource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Type      string `json:"type"`
	Method    string `json:"method"`
	Headers   string `json:"headers"`
	Path      string `json:"path"`
	Interval  int    `json:"interval"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

type OracleData struct {
	ID          string `json:"id"`
	SourceID    string `json:"source_id"`
	Value       string `json:"value"`
	RawResponse string `json:"raw_response"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type Storage interface {
	SaveDataSource(ds *DataSource) error
	GetDataSource(id string) (*DataSource, error)
	ListDataSources() ([]*DataSource, error)
	UpdateDataSource(ds *DataSource) error
	DeleteDataSource(id string) error

	SaveOracleData(d *OracleData) error
	GetOracleData(id string) (*OracleData, error)
	GetOracleDataBySource(sourceID string, limit int) ([]*OracleData, error)
	GetLatestOracleData(sourceID string) (*OracleData, error)
	GetOracleDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error)

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

func (s *SQLiteStorage) initTables() error {
	if _, err := s.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS data_sources (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			type TEXT,
			method TEXT DEFAULT 'GET',
			headers TEXT,
			path TEXT,
			interval INTEGER DEFAULT 60,
			enabled INTEGER DEFAULT 1,
			created_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS oracle_data (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			value TEXT,
			raw_response TEXT,
			timestamp INTEGER,
			block_height INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_oracle_data_source ON oracle_data(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_oracle_data_timestamp ON oracle_data(timestamp)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to exec query: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStorage) SaveDataSource(ds *DataSource) error {
	if ds.ID == "" {
		ds.ID = uuid.New().String()
	}
	if ds.CreatedAt == 0 {
		ds.CreatedAt = time.Now().Unix()
	}
	if ds.Method == "" {
		ds.Method = "GET"
	}
	enabled := 0
	if ds.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO data_sources (id, name, url, type, method, headers, path, interval, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ds.ID, ds.Name, ds.URL, ds.Type, ds.Method, ds.Headers, ds.Path, ds.Interval, enabled, ds.CreatedAt,
	)
	return err
}

func (s *SQLiteStorage) GetDataSource(id string) (*DataSource, error) {
	row := s.db.QueryRow(
		`SELECT id, name, url, type, method, headers, path, interval, enabled, created_at FROM data_sources WHERE id = ?`,
		id,
	)
	ds := &DataSource{}
	var enabled int
	err := row.Scan(&ds.ID, &ds.Name, &ds.URL, &ds.Type, &ds.Method, &ds.Headers, &ds.Path, &ds.Interval, &enabled, &ds.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	ds.Enabled = enabled == 1
	return ds, err
}

func (s *SQLiteStorage) ListDataSources() ([]*DataSource, error) {
	rows, err := s.db.Query(
		`SELECT id, name, url, type, method, headers, path, interval, enabled, created_at FROM data_sources ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*DataSource
	for rows.Next() {
		ds := &DataSource{}
		var enabled int
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.URL, &ds.Type, &ds.Method, &ds.Headers, &ds.Path, &ds.Interval, &enabled, &ds.CreatedAt); err != nil {
			return nil, err
		}
		ds.Enabled = enabled == 1
		sources = append(sources, ds)
	}
	return sources, rows.Err()
}

func (s *SQLiteStorage) UpdateDataSource(ds *DataSource) error {
	enabled := 0
	if ds.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(
		`UPDATE data_sources SET name = ?, url = ?, type = ?, method = ?, headers = ?, path = ?, interval = ?, enabled = ? WHERE id = ?`,
		ds.Name, ds.URL, ds.Type, ds.Method, ds.Headers, ds.Path, ds.Interval, enabled, ds.ID,
	)
	return err
}

func (s *SQLiteStorage) DeleteDataSource(id string) error {
	_, err := s.db.Exec(`DELETE FROM data_sources WHERE id = ?`, id)
	return err
}

func (s *SQLiteStorage) SaveOracleData(d *OracleData) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.Timestamp == 0 {
		d.Timestamp = time.Now().Unix()
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO oracle_data (id, source_id, value, raw_response, timestamp, block_height)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		d.ID, d.SourceID, d.Value, d.RawResponse, d.Timestamp, d.BlockHeight,
	)
	return err
}

func (s *SQLiteStorage) GetOracleData(id string) (*OracleData, error) {
	row := s.db.QueryRow(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE id = ?`,
		id,
	)
	d := &OracleData{}
	err := row.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (s *SQLiteStorage) GetOracleDataBySource(sourceID string, limit int) ([]*OracleData, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE source_id = ? ORDER BY timestamp DESC LIMIT ?`,
		sourceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []*OracleData
	for rows.Next() {
		d := &OracleData{}
		if err := rows.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight); err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (s *SQLiteStorage) GetLatestOracleData(sourceID string) (*OracleData, error) {
	row := s.db.QueryRow(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE source_id = ? ORDER BY timestamp DESC LIMIT 1`,
		sourceID,
	)
	d := &OracleData{}
	err := row.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (s *SQLiteStorage) GetOracleDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error) {
	rows, err := s.db.Query(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE source_id = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp DESC`,
		sourceID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []*OracleData
	for rows.Next() {
		d := &OracleData{}
		if err := rows.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight); err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
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
	dataSources map[string]*DataSource
	oracleData  map[string]*OracleData
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		dataSources: make(map[string]*DataSource),
		oracleData:  make(map[string]*OracleData),
	}
}

func (s *InMemoryStorage) SaveDataSource(ds *DataSource) error {
	if ds.ID == "" {
		ds.ID = uuid.New().String()
	}
	if ds.CreatedAt == 0 {
		ds.CreatedAt = time.Now().Unix()
	}
	s.dataSources[ds.ID] = ds
	return nil
}

func (s *InMemoryStorage) GetDataSource(id string) (*DataSource, error) {
	return s.dataSources[id], nil
}

func (s *InMemoryStorage) ListDataSources() ([]*DataSource, error) {
	var sources []*DataSource
	for _, ds := range s.dataSources {
		sources = append(sources, ds)
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].CreatedAt > sources[j].CreatedAt
	})
	return sources, nil
}

func (s *InMemoryStorage) UpdateDataSource(ds *DataSource) error {
	s.dataSources[ds.ID] = ds
	return nil
}

func (s *InMemoryStorage) DeleteDataSource(id string) error {
	delete(s.dataSources, id)
	return nil
}

func (s *InMemoryStorage) SaveOracleData(d *OracleData) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.Timestamp == 0 {
		d.Timestamp = time.Now().Unix()
	}
	s.oracleData[d.ID] = d
	return nil
}

func (s *InMemoryStorage) GetOracleData(id string) (*OracleData, error) {
	return s.oracleData[id], nil
}

func (s *InMemoryStorage) GetOracleDataBySource(sourceID string, limit int) ([]*OracleData, error) {
	if limit <= 0 {
		limit = 100
	}
	var data []*OracleData
	for _, d := range s.oracleData {
		if d.SourceID == sourceID {
			data = append(data, d)
		}
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp > data[j].Timestamp
	})
	if len(data) > limit {
		data = data[:limit]
	}
	return data, nil
}

func (s *InMemoryStorage) GetLatestOracleData(sourceID string) (*OracleData, error) {
	var latest *OracleData
	var latestTs int64
	for _, d := range s.oracleData {
		if d.SourceID == sourceID && d.Timestamp > latestTs {
			latest = d
			latestTs = d.Timestamp
		}
	}
	return latest, nil
}

func (s *InMemoryStorage) GetOracleDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error) {
	var data []*OracleData
	for _, d := range s.oracleData {
		if d.SourceID == sourceID && d.Timestamp >= start && d.Timestamp <= end {
			data = append(data, d)
		}
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp > data[j].Timestamp
	})
	return data, nil
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
