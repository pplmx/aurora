package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	versions := make([]int, 0, len(byVersion))
	for version := range byVersion {
		versions = append(versions, int(version))
	}
	sort.Ints(versions)
	result := make([]migration, 0, len(versions))
	for _, version := range versions {
		item := byVersion[uint(version)]
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

func (m *Migrator) setVersion(version uint, dirty bool) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM schema_migrations`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if version > 0 || dirty {
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES (?, ?)`, version, dirty); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// run marks the target version dirty before applying the SQL, then clears the
// bit in the same successful path. A failure leaves the database visibly dirty
// so neither up nor down will retry a partially applied schema.
func (m *Migrator) run(version uint, body string) error {
	if err := m.setVersion(version, true); err != nil {
		return err
	}
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(body); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return m.setVersion(version, false)
}

func (m *Migrator) up(steps int) (uint, error) {
	items, err := m.migrations()
	if err != nil {
		return 0, err
	}
	current, dirty, err := m.version()
	if err != nil {
		return 0, err
	}
	if dirty {
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
		if err := m.run(item.version, item.up); err != nil {
			return current, fmt.Errorf("migration %d up failed: %w", item.version, err)
		}
		current = item.version
		applied++
	}
	if applied == 0 {
		return current, errNoChange
	}
	return current, nil
}

func (m *Migrator) down(steps int) (uint, error) {
	items, err := m.migrations()
	if err != nil {
		return 0, err
	}
	current, dirty, err := m.version()
	if err != nil {
		return 0, err
	}
	if dirty {
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
			return current, fmt.Errorf("migration %d has no down file", item.version)
		}
		if err := m.run(item.version-1, item.down); err != nil {
			return current, fmt.Errorf("migration %d down failed: %w", item.version, err)
		}
		current = item.version - 1
		applied++
	}
	if applied == 0 {
		return current, errNoChange
	}
	return current, nil
}
