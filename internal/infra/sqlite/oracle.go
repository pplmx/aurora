package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/oracle"
)

type OracleRepository struct {
	db *sql.DB
}

func NewOracleRepository(path string) (*OracleRepository, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &OracleRepository{db: db}
	if err := repo.initTables(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}
	return repo, nil
}

func (r *OracleRepository) initTables() error {
	if _, err := r.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
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
		if _, err := r.db.Exec(query); err != nil {
			return fmt.Errorf("failed to exec query: %w", err)
		}
	}
	return nil
}

func (r *OracleRepository) SaveData(data *oracle.OracleData) error {
	if data.ID == "" {
		data.ID = uuid.New().String()
	}
	if data.Timestamp == 0 {
		data.Timestamp = time.Now().Unix()
	}
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO oracle_data (id, source_id, value, raw_response, timestamp, block_height)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		data.ID, data.SourceID, data.Value, data.RawResponse, data.Timestamp, data.BlockHeight,
	)
	return err
}

func (r *OracleRepository) GetData(id string) (*oracle.OracleData, error) {
	row := r.db.QueryRow(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE id = ?`,
		id,
	)
	d := &oracle.OracleData{}
	err := row.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (r *OracleRepository) GetDataBySource(sourceID string, limit int) ([]*oracle.OracleData, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE source_id = ? ORDER BY timestamp DESC LIMIT ?`,
		sourceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var data []*oracle.OracleData
	for rows.Next() {
		d := &oracle.OracleData{}
		if err := rows.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight); err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (r *OracleRepository) GetLatestData(sourceID string) (*oracle.OracleData, error) {
	row := r.db.QueryRow(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE source_id = ? ORDER BY timestamp DESC LIMIT 1`,
		sourceID,
	)
	d := &oracle.OracleData{}
	err := row.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (r *OracleRepository) GetDataByTimeRange(sourceID string, start, end int64) ([]*oracle.OracleData, error) {
	rows, err := r.db.Query(
		`SELECT id, source_id, value, raw_response, timestamp, block_height FROM oracle_data WHERE source_id = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp DESC`,
		sourceID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var data []*oracle.OracleData
	for rows.Next() {
		d := &oracle.OracleData{}
		if err := rows.Scan(&d.ID, &d.SourceID, &d.Value, &d.RawResponse, &d.Timestamp, &d.BlockHeight); err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, rows.Err()
}

func (r *OracleRepository) SaveSource(source *oracle.DataSource) error {
	if source.ID == "" {
		source.ID = uuid.New().String()
	}
	if source.CreatedAt == 0 {
		source.CreatedAt = time.Now().Unix()
	}
	if source.Method == "" {
		source.Method = "GET"
	}
	enabled := 0
	if source.Enabled {
		enabled = 1
	}
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO data_sources (id, name, url, type, method, headers, path, interval, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		source.ID, source.Name, source.URL, source.Type, source.Method, source.Headers, source.Path, source.Interval, enabled, source.CreatedAt,
	)
	return err
}

func (r *OracleRepository) GetSource(id string) (*oracle.DataSource, error) {
	row := r.db.QueryRow(
		`SELECT id, name, url, type, method, headers, path, interval, enabled, created_at FROM data_sources WHERE id = ?`,
		id,
	)
	ds := &oracle.DataSource{}
	var enabled int
	err := row.Scan(&ds.ID, &ds.Name, &ds.URL, &ds.Type, &ds.Method, &ds.Headers, &ds.Path, &ds.Interval, &enabled, &ds.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	ds.Enabled = enabled == 1
	return ds, err
}

func (r *OracleRepository) ListSources() ([]*oracle.DataSource, error) {
	rows, err := r.db.Query(
		`SELECT id, name, url, type, method, headers, path, interval, enabled, created_at FROM data_sources ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sources []*oracle.DataSource
	for rows.Next() {
		ds := &oracle.DataSource{}
		var enabled int
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.URL, &ds.Type, &ds.Method, &ds.Headers, &ds.Path, &ds.Interval, &enabled, &ds.CreatedAt); err != nil {
			return nil, err
		}
		ds.Enabled = enabled == 1
		sources = append(sources, ds)
	}
	return sources, rows.Err()
}

func (r *OracleRepository) UpdateSource(ds *oracle.DataSource) error {
	enabled := 0
	if ds.Enabled {
		enabled = 1
	}
	_, err := r.db.Exec(
		`UPDATE data_sources SET name = ?, url = ?, type = ?, method = ?, headers = ?, path = ?, interval = ?, enabled = ? WHERE id = ?`,
		ds.Name, ds.URL, ds.Type, ds.Method, ds.Headers, ds.Path, ds.Interval, enabled, ds.ID,
	)
	return err
}

func (r *OracleRepository) DeleteSource(id string) error {
	_, err := r.db.Exec(`DELETE FROM data_sources WHERE id = ?`, id)
	return err
}

type InMemoryOracleRepository struct {
	dataSources map[string]*oracle.DataSource
	oracleData  map[string]*oracle.OracleData
}

func NewInMemoryOracleRepository() *InMemoryOracleRepository {
	return &InMemoryOracleRepository{
		dataSources: make(map[string]*oracle.DataSource),
		oracleData:  make(map[string]*oracle.OracleData),
	}
}

func (r *InMemoryOracleRepository) SaveData(data *oracle.OracleData) error {
	if data.ID == "" {
		data.ID = uuid.New().String()
	}
	if data.Timestamp == 0 {
		data.Timestamp = time.Now().Unix()
	}
	r.oracleData[data.ID] = data
	return nil
}

func (r *InMemoryOracleRepository) GetData(id string) (*oracle.OracleData, error) {
	return r.oracleData[id], nil
}

func (r *InMemoryOracleRepository) GetDataBySource(sourceID string, limit int) ([]*oracle.OracleData, error) {
	if limit <= 0 {
		limit = 100
	}
	var data []*oracle.OracleData
	for _, d := range r.oracleData {
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

func (r *InMemoryOracleRepository) GetLatestData(sourceID string) (*oracle.OracleData, error) {
	var latest *oracle.OracleData
	var latestTs int64
	for _, d := range r.oracleData {
		if d.SourceID == sourceID && d.Timestamp > latestTs {
			latest = d
			latestTs = d.Timestamp
		}
	}
	return latest, nil
}

func (r *InMemoryOracleRepository) GetDataByTimeRange(sourceID string, start, end int64) ([]*oracle.OracleData, error) {
	var data []*oracle.OracleData
	for _, d := range r.oracleData {
		if d.SourceID == sourceID && d.Timestamp >= start && d.Timestamp <= end {
			data = append(data, d)
		}
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp > data[j].Timestamp
	})
	return data, nil
}

func (r *InMemoryOracleRepository) SaveSource(source *oracle.DataSource) error {
	if source.ID == "" {
		source.ID = uuid.New().String()
	}
	if source.CreatedAt == 0 {
		source.CreatedAt = time.Now().Unix()
	}
	r.dataSources[source.ID] = source
	return nil
}

func (r *InMemoryOracleRepository) GetSource(id string) (*oracle.DataSource, error) {
	return r.dataSources[id], nil
}

func (r *InMemoryOracleRepository) ListSources() ([]*oracle.DataSource, error) {
	var sources []*oracle.DataSource
	for _, ds := range r.dataSources {
		sources = append(sources, ds)
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].CreatedAt > sources[j].CreatedAt
	})
	return sources, nil
}

func (r *InMemoryOracleRepository) UpdateSource(ds *oracle.DataSource) error {
	r.dataSources[ds.ID] = ds
	return nil
}

func (r *InMemoryOracleRepository) DeleteSource(id string) error {
	delete(r.dataSources, id)
	return nil
}
