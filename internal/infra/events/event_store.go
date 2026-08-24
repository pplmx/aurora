package events

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/events"
)

type EventRepository interface {
	Save(event events.Event) error
	GetByType(eventType string, limit int) ([]events.Event, error)
	GetByModule(module string, limit int) ([]events.Event, error)
	GetByAggregate(aggID string, limit, offset int) ([]events.Event, error)
	// GetByAggregateAndType pages over events of ONE type within an aggregate,
	// applying LIMIT/OFFSET at the SQL layer over those matching rows only.
	// GetByAggregate alone counts ALL event types in the aggregate before any
	// consumer-side type filter, so paging transfer history through it
	// under-fills when a token has interleaved mint/burn events (TASK-074,
	// ISS-066). Filtering type first restores correct page composition.
	GetByAggregateAndType(aggID, eventType string, limit, offset int) ([]events.Event, error)
}

type SQLiteEventStore struct {
	db *sql.DB
}

func NewSQLiteEventStore(dbPath string) (*SQLiteEventStore, error) {
	// Writes are autocommit single INSERTs (no explicit transactions), so
	// _txlock is not needed here; _busy_timeout makes a concurrent publish wait
	// for the WAL write lock over the API pool instead of failing with
	// SQLITE_BUSY (v1.70, ISS-076 — same value as the v1.67 replay DSN).
	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON&_busy_timeout=5000", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	es := &SQLiteEventStore{db: database}

	if err := es.createTables(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return es, nil
}

func (e *SQLiteEventStore) createTables() error {
	if _, err := e.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := e.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id          TEXT PRIMARY KEY,
			event_type  TEXT NOT NULL,
			module      TEXT NOT NULL,
			agg_id      TEXT NOT NULL,
			payload     BLOB NOT NULL,
			timestamp   INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_events_module ON events(module)`,
		`CREATE INDEX IF NOT EXISTS idx_events_agg ON events(agg_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC)`,
	}

	for _, query := range queries {
		if _, err := e.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (e *SQLiteEventStore) Save(event events.Event) error {
	_, err := e.db.Exec(`
		INSERT INTO events (id, event_type, module, agg_id, payload, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID(), event.EventType(), event.Module(), event.AggregateID(), event.Payload(), event.Timestamp().Unix())
	return err
}

func (e *SQLiteEventStore) GetByType(eventType string, limit int) ([]events.Event, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := e.db.Query(`
		SELECT id, event_type, module, agg_id, payload, timestamp
		FROM events
		WHERE event_type = ?
		ORDER BY timestamp DESC, id DESC
		LIMIT ?
	`, eventType, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanEvents(rows)
}

func (e *SQLiteEventStore) GetByModule(module string, limit int) ([]events.Event, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := e.db.Query(`
		SELECT id, event_type, module, agg_id, payload, timestamp
		FROM events
		WHERE module = ?
		ORDER BY timestamp DESC, id DESC
		LIMIT ?
	`, module, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanEvents(rows)
}

func (e *SQLiteEventStore) GetByAggregate(aggID string, limit, offset int) ([]events.Event, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := e.db.Query(`
		SELECT id, event_type, module, agg_id, payload, timestamp
		FROM events
		WHERE agg_id = ?
		ORDER BY timestamp ASC, id ASC
		LIMIT ? OFFSET ?
	`, aggID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanEvents(rows)
}

// GetByAggregateAndType is like GetByAggregate but restricts to a single event
// type so LIMIT/OFFSET are applied over the matching rows (e.g. transfers)
// rather than every event in the aggregate (transfers + mints + burns). This is
// the pagination primitive the token transfer-history reader needs.
func (e *SQLiteEventStore) GetByAggregateAndType(aggID, eventType string, limit, offset int) ([]events.Event, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := e.db.Query(`
		SELECT id, event_type, module, agg_id, payload, timestamp
		FROM events
		WHERE agg_id = ? AND event_type = ?
		ORDER BY timestamp ASC, id ASC
		LIMIT ? OFFSET ?
	`, aggID, eventType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanEvents(rows)
}

func scanEvents(rows *sql.Rows) ([]events.Event, error) {
	var result []events.Event
	for rows.Next() {
		var id, eventType, module, aggID string
		var payload []byte
		var timestamp int64

		if err := rows.Scan(&id, &eventType, &module, &aggID, &payload, &timestamp); err != nil {
			return nil, err
		}

		event := events.NewStoredEvent(id, time.Unix(timestamp, 0), eventType, aggID, payload)
		result = append(result, event)
	}
	return result, rows.Err()
}

func (e *SQLiteEventStore) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}
