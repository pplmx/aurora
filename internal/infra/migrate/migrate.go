package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/utils"
	_ "modernc.org/sqlite"
)

type Migrator struct {
	db      *sql.DB
	migPath string
}

type MigrationStatus struct {
	Current         uint
	Dirty           bool
	Applied         []uint
	PendingVersions []uint
}

func New(dbPath, migPath string) (*Migrator, error) {
	if err := utils.EnsureSQLiteDirectory(dbPath); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Use the same hardened DSN every repository opens (foreign keys + BEGIN
	// IMMEDIATE + 5s busy timeout, sqlite.WriteDSNSuffix) instead of a bare
	// `?_foreign_keys=ON`. An unhardened migration against a DB another
	// CLI/API process holds the write lock on otherwise fails immediately with
	// "database is locked" (v1.70, ISS-076; TASK-170, ISS-165).
	db, err := sql.Open("sqlite", sqlite.DSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	m := &Migrator{db: db, migPath: migPath}
	if err := m.ensureVersionTable(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create migration table: %w", err)
	}
	if _, err := m.migrations(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to open migration source: %w", err)
	}
	if err := utils.RestrictSQLiteFiles(dbPath); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to secure database files: %w", err)
	}
	return m, nil
}

func (m *Migrator) ensureVersionTable() error {
	if _, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + migrationsTable + ` (version INTEGER PRIMARY KEY, dirty INTEGER NOT NULL DEFAULT 0);
		CREATE UNIQUE INDEX IF NOT EXISTS version_unique ON ` + migrationsTable + `(version);
	`); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) Up(steps int) (uint, error) {
	version, err := m.up(steps)
	if errors.Is(err, errNoChange) {
		return version, nil
	}
	if err != nil {
		return 0, fmt.Errorf("migration up failed: %w", explainBusy(err))
	}
	return version, nil
}

func (m *Migrator) Down(steps int) (uint, error) {
	version, err := m.down(steps)
	if errors.Is(err, errNoChange) {
		return version, nil
	}
	if err != nil {
		return 0, fmt.Errorf("migration down failed: %w", explainBusy(err))
	}
	return version, nil
}

// explainBusy turns SQLite's bare lock error into something an operator can act
// on. A migration holds the write lock from its first statement to commit, so a
// concurrent CLI or API process still writing makes it wait up to the DSN's
// busy timeout and then fail with "database is locked (5) (SQLITE_BUSY)" —
// measured: an 8-second concurrent writer against the 5-second timeout fails
// the migration after 5s, and the database is left untouched, so a retry
// succeeds once the writer finishes.
//
// Every other error is returned unchanged.
func explainBusy(err error) error {
	if err == nil || !isBusy(err) {
		return err
	}
	return fmt.Errorf(
		"%w — another process is writing this database; stop it (or wait for it) and retry. The database was not modified",
		err,
	)
}

// isBusy reports whether err is SQLite's "database is locked" / SQLITE_BUSY.
// The driver wraps the code in its message, so the text is what identifies it;
// errors.Is has nothing to match on.
func isBusy(err error) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		msg := e.Error()
		if strings.Contains(msg, "database is locked") || strings.Contains(msg, "SQLITE_BUSY") {
			return true
		}
	}
	return false
}

// Repair clears the dirty marker left by a failed migration, after the
// operator has checked the schema. A dirty database refuses both Up and Down —
// by design, because the runner cannot know whether a half-applied body
// committed — so without this the only way out of a failed migration was to
// delete the database and lose every record in it.
//
// It deliberately does not roll anything back or re-run a migration: it only
// clears the flag. The caller decides what the schema is at, which is why the
// CLI asks for --confirm and prints the version it cleared.
func (m *Migrator) Repair() (uint, error) {
	current, dirty, err := m.version()
	if err != nil {
		return 0, fmt.Errorf("failed to get version: %w", err)
	}
	if !dirty {
		return current, errNotDirty
	}
	if err := setTxVersionTx(m.db, current, false); err != nil {
		return 0, fmt.Errorf("failed to clear the dirty marker: %w", err)
	}
	return current, nil
}

// errNotDirty reports a repair of a database that is not dirty, so the CLI can
// say "nothing to repair" instead of pretending it fixed something.
var errNotDirty = errors.New("database is not dirty")

// ErrNotDirty exposes errNotDirty to callers that report it.
func ErrNotDirty() error { return errNotDirty }

// setTxVersionTx writes the migration row outside a transaction, for the
// repair path that has none.
func setTxVersionTx(db *sql.DB, version uint, dirty bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := setTxVersion(tx, version, dirty); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (m *Migrator) Status() (*MigrationStatus, error) {
	items, err := m.migrations()
	if err != nil {
		return nil, err
	}
	current, dirty, err := m.version()
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	applied := make([]uint, 0)
	pending := make([]uint, 0)
	for _, item := range items {
		if item.version <= current {
			applied = append(applied, item.version)
		} else {
			pending = append(pending, item.version)
		}
	}
	return &MigrationStatus{
		Current:         current,
		Dirty:           dirty,
		Applied:         applied,
		PendingVersions: pending,
	}, nil
}

func (m *Migrator) Version() (uint, error) {
	version, _, err := m.version()
	return version, err
}

func (m *Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

type MigrateConfig struct {
	AutoMigrate bool
	MigPath     string
}

func RunMigrationsIfEnabled(dbPath string, cfg MigrateConfig) error {
	if !cfg.AutoMigrate {
		return nil
	}
	migPath := cfg.MigPath
	if migPath == "" {
		migPath = "./migrations"
	}
	m, err := New(dbPath, migPath)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() { _ = m.Close() }()
	if _, err := m.Up(0); err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}
	return nil
}
