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
	// GetByAggregateAndTypePayload pages over the aggregate's events of one
	// type AND whose payload field at payloadPath equals value, applying
	// LIMIT/OFFSET over those matching rows only. This is the owner-scoped
	// pagination primitive token transfer history needs (TASK-093, ISS-086).
	GetByAggregateAndTypePayload(aggID, eventType, payloadPath, value string, limit, offset int) ([]events.Event, error)
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
		// pending_events is the durable outbox for audit events whose direct
		// delivery INSERT failed (TASK-119, ISS-111). The PK is the event's own
		// id (a random UUID), so EnqueuePending is idempotent and a drainer retry
		// that lands in events can never duplicate rows (the same PK constraint
		// makes SaveIdempotent's INSERT OR IGNORE the UNIQUE-safe retry).
		`CREATE TABLE IF NOT EXISTS pending_events (
			id              TEXT PRIMARY KEY,
			event_type      TEXT NOT NULL,
			module          TEXT NOT NULL,
			agg_id          TEXT NOT NULL,
			payload         BLOB NOT NULL,
			timestamp       INTEGER NOT NULL,
			attempts        INTEGER NOT NULL DEFAULT 0,
			next_attempt_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pending_next ON pending_events(next_attempt_at)`,
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

// SaveIdempotent is the UNIQUE-safe retry path for the outbox drainer
// (TASK-119, ISS-111): INSERT OR IGNORE keyed on the event id, so a retried
// publish that already landed in events — from a previous attempt whose
// response was lost, or from the time between outbox enqueue and drain — never
// errors on the PK conflict. The direct delivery path keeps plain Save (a
// duplicate direct publish is a programming error worth surfacing); only the
// retry loop needs idempotency.
func (e *SQLiteEventStore) SaveIdempotent(event events.Event) error {
	_, err := e.db.Exec(`
		INSERT OR IGNORE INTO events (id, event_type, module, agg_id, payload, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID(), event.EventType(), event.Module(), event.AggregateID(), event.Payload(), event.Timestamp().Unix())
	return err
}

// PendingEvent is one row of the durable outbox (pending_events): an audit
// event whose direct delivery failed, waiting for the drainer to retry.
type PendingEvent struct {
	Event     events.Event
	Attempts  int
	NextTryAt time.Time
}

// EnqueuePending durably records a failed delivery into the outbox. It is
// idempotent on the event id (UUID): re-enqueueing an event already waiting
// (e.g. a retried publish while the drainer lagged) is a no-op, never a
// duplicate. Called by AuditHandler when the direct Save fails (TASK-119).
func (e *SQLiteEventStore) EnqueuePending(event events.Event) error {
	_, err := e.db.Exec(`
		INSERT OR IGNORE INTO pending_events (
			id, event_type, module, agg_id, payload, timestamp,
			attempts, next_attempt_at
		) VALUES (?, ?, ?, ?, ?, ?, 0, ?)
	`, event.ID(), event.EventType(), event.Module(), event.AggregateID(), event.Payload(),
		event.Timestamp().Unix(), event.Timestamp().Unix())
	return err
}

// ListDuePending returns the outbox rows whose next_attempt_at has elapsed,
// oldest next-try first, capped at limit. The drainer calls this, retries each
// via SaveIdempotent, and drops the rows that landed. limit <= 0 defaults to a
// modest batch (50) so a large backlog drains in waves without one giant read.
func (e *SQLiteEventStore) ListDuePending(now time.Time, limit int) ([]PendingEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := e.db.Query(`
		SELECT id, event_type, module, agg_id, payload, timestamp, attempts, next_attempt_at
		FROM pending_events
		WHERE next_attempt_at <= ?
		ORDER BY next_attempt_at ASC
		LIMIT ?
	`, now.Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []PendingEvent
	for rows.Next() {
		var id, eventType, module, aggID string
		var payload []byte
		var timestamp, attempts, nextTryAt int64
		if err := rows.Scan(&id, &eventType, &module, &aggID, &payload, &timestamp, &attempts, &nextTryAt); err != nil {
			return nil, err
		}
		result = append(result, PendingEvent{
			Event:     events.NewStoredEvent(id, time.Unix(timestamp, 0), eventType, aggID, payload),
			Attempts:  int(attempts),
			NextTryAt: time.Unix(nextTryAt, 0),
		})
	}
	return result, rows.Err()
}

// DropPending removes a pending_events row, either because the retry landed in
// events (SaveIdempotent succeeded) or because the outbox entry was explicitly
// cleared. It tolerates an already-absent id (IsNotExist is not an error).
func (e *SQLiteEventStore) DropPending(id string) error {
	_, err := e.db.Exec(`DELETE FROM pending_events WHERE id = ?`, id)
	return err
}

// BackoffPending records the next_attempt_at for a failed retry: attempts is
// the incremented count, nextTry is when the drainer may try again. This is the
// only growth a stuck source causes — a permanently-failing destination's
// pending row backs off geometrically (bounded by the drainer's cap) instead of
// hammering the events table every drain tick.
func (e *SQLiteEventStore) BackoffPending(id string, attempts int, nextTry time.Time) error {
	_, err := e.db.Exec(`
		UPDATE pending_events SET attempts = ?, next_attempt_at = ?
		WHERE id = ?
	`, attempts, nextTry.Unix(), id)
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

// GetByAggregateAndTypePayload extends GetByAggregateAndType with a JSON-path
// predicate over the event payload, so LIMIT/OFFSET are applied over the rows
// matching BOTH the type and the payload field. Token transfer history needs
// this for the owner dimension: paging over all token.transfer events and then
// filtering the requested owner in memory under-fills every page on a
// multi-owner token and steps offset through the whole token stream instead of
// the owner's (TASK-093, ISS-086). payloadPath is a SQLite JSON path such as
// "$.from"; payload is stored as BLOB, so it is cast to TEXT for json_extract.
func (e *SQLiteEventStore) GetByAggregateAndTypePayload(aggID, eventType, payloadPath, value string, limit, offset int) ([]events.Event, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := e.db.Query(`
		SELECT id, event_type, module, agg_id, payload, timestamp
		FROM events
		WHERE agg_id = ? AND event_type = ?
		  AND json_extract(CAST(payload AS TEXT), ?) = ?
		ORDER BY timestamp ASC, id ASC
		LIMIT ? OFFSET ?
	`, aggID, eventType, payloadPath, value, limit, offset)
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
