package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const migrationsTable = "schema_migrations"

var errNoChange = errors.New("no change")

type migration struct {
	version uint
	up      string
	down    string
}

// migrations reads every versioned SQL file once and returns them in version
// order. A malformed filename is ignored like a README, but an incomplete or
// ambiguous version is rejected: migrations must never depend on map order.
func (m *Migrator) migrations() ([]migration, error) {
	entries, err := os.ReadDir(m.migPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	byVersion := make(map[uint]*migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version, direction, ok := parseMigrationName(entry.Name())
		if !ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join(m.migPath, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}
		item := byVersion[version]
		if item == nil {
			item = &migration{version: version}
			byVersion[version] = item
		}
		switch direction {
		case "up":
			if item.up != "" {
				return nil, fmt.Errorf("duplicate up migration for version %d", version)
			}
			item.up = string(data)
		case "down":
			if item.down != "" {
				return nil, fmt.Errorf("duplicate down migration for version %d", version)
			}
			item.down = string(data)
		}
	}

	// Sorted map keys: migration order must never depend on map iteration order.
	result := make([]migration, 0, len(byVersion))
	for _, version := range slices.Sorted(maps.Keys(byVersion)) {
		item := byVersion[version]
		if item.up == "" {
			return nil, fmt.Errorf("migration %06d has no up file", version)
		}
		result = append(result, *item)
	}
	return result, nil
}

func parseMigrationName(name string) (uint, string, bool) {
	base, ok := strings.CutSuffix(name, ".sql")
	if !ok {
		return 0, "", false
	}
	direction := ""
	switch {
	case strings.HasSuffix(base, ".up"):
		direction = "up"
	case strings.HasSuffix(base, ".down"):
		direction = "down"
	default:
		return 0, "", false
	}
	versionText := base[:len(base)-len(direction)-1]
	versionEnd := strings.IndexByte(versionText, '_')
	if versionEnd < 0 {
		versionEnd = len(versionText)
	}
	version, err := strconv.ParseUint(versionText[:versionEnd], 10, 64)
	if err != nil || version == 0 {
		return 0, "", false
	}
	return uint(version), direction, true
}

func (m *Migrator) version() (uint, bool, error) {
	var version uint
	var dirty bool
	err := m.db.QueryRow(`SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return version, dirty, err
}

// txVersion reads the committed/current transactional migration state. It is
// called only while already holding the database write lock, so concurrent
// migrators cannot observe an in-flight dirty marker.
func txVersion(q queryer) (uint, bool, error) {
	var version uint
	var dirty bool
	err := q.QueryRow(`SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return version, dirty, err
}

func setTxVersion(tx *sql.Tx, version uint, dirty bool) error {
	if _, err := tx.Exec(`DELETE FROM schema_migrations`); err != nil {
		return err
	}
	if version > 0 || dirty {
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES (?, ?)`, version, dirty); err != nil {
			return err
		}
	}
	return nil
}

// applyTx runs one migration body and records the version the schema is at
// afterwards. `version` identifies the migration being applied or rolled back
// and names the savepoint; `target` is the resulting schema version, which for
// a rollback is version-1 (the migration the schema no longer carries).
//
// The savepoint keeps a failed body from poisoning the outer transaction: the
// body is undone on its own, so earlier steps in the same run survive and the
// caller can still commit a dirty marker for the attempt.
func applyTx(tx *sql.Tx, version, target uint, body string) error {
	if _, err := tx.Exec(fmt.Sprintf("SAVEPOINT migration_%d", version)); err != nil {
		return err
	}
	if _, err := tx.Exec(body); err != nil {
		if _, rollbackErr := tx.Exec(fmt.Sprintf("ROLLBACK TO migration_%d", version)); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf("RELEASE migration_%d", version)); err != nil {
		return err
	}
	return setTxVersion(tx, target, false)
}

// markFailedTx durably records a failed migration's target version as dirty
// and commits that marker. The body has already been rolled back to its
// savepoint, so the schema is consistent — the marker exists to make the next
// run refuse rather than silently repeat an attempt whose outcome is unknown.
// Returns nil when the marker is committed.
func markFailedTx(tx *sql.Tx, target uint) error {
	if err := setTxVersion(tx, target, true); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("could not record the failed migration as dirty: %w", err)
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("could not record the failed migration as dirty: %w", err)
	}
	return nil
}

func (m *Migrator) up(steps int) (uint, error) {
	items, err := m.migrations()
	if err != nil {
		return 0, err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	current, dirty, err := txVersion(tx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if dirty {
		_ = tx.Rollback()
		return current, fmt.Errorf("database is dirty at version %d", current)
	}

	applied := 0
	for _, item := range items {
		if item.version <= current {
			continue
		}
		if steps > 0 && applied >= steps {
			break
		}
		bodyErr := applyTx(tx, item.version, item.version, item.up)
		if bodyErr != nil {
			// The body was undone on its own savepoint, so the migrations that
			// already ran in this transaction are intact. Record the attempt as
			// dirty at its target version and commit that marker; a later run
			// must refuse instead of guessing whether it partially applied.
			if markErr := markFailedTx(tx, item.version); markErr != nil {
				return current, errors.Join(
					fmt.Errorf("migration %d up failed: %w", item.version, bodyErr), markErr)
			}
			return current, fmt.Errorf("migration %d up failed: %w", item.version, bodyErr)
		}
		current = item.version
		applied++
	}

	if applied == 0 {
		// Nothing to do, but the transaction is already open and holds the write
		// lock: release it instead of leaving it for the connection pool to GC.
		_ = tx.Rollback()
		return current, errNoChange
	}
	return current, tx.Commit()
}

func (m *Migrator) down(steps int) (uint, error) {
	items, err := m.migrations()
	if err != nil {
		return 0, err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	current, dirty, err := txVersion(tx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if dirty {
		_ = tx.Rollback()
		return current, fmt.Errorf("database is dirty at version %d", current)
	}

	applied := 0
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.version > current {
			continue
		}
		if steps > 0 && applied >= steps {
			break
		}
		if item.down == "" {
			_ = tx.Rollback()
			return current, fmt.Errorf("migration %d has no down file", item.version)
		}
		// Rolling back migration N leaves the schema at N-1, so that is the
		// version to record on success and the one to mark dirty on failure —
		// the same "target version" rule the up loop uses.
		target := item.version - 1
		bodyErr := applyTx(tx, item.version, target, item.down)
		if bodyErr != nil {
			if markErr := markFailedTx(tx, target); markErr != nil {
				return current, errors.Join(
					fmt.Errorf("migration %d down failed: %w", item.version, bodyErr), markErr)
			}
			return current, fmt.Errorf("migration %d down failed: %w", item.version, bodyErr)
		}
		current = target
		applied++
	}
	if applied == 0 {
		_ = tx.Rollback()
		return current, errNoChange
	}
	return current, tx.Commit()
}

// queryer is the subset of *sql.Tx used to read the transactional migration
// state without coupling helper functions to the concrete transaction type.
type queryer interface {
	QueryRow(query string, args ...any) *sql.Row
}
