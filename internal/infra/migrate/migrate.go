package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pplmx/aurora/internal/infra/sqlite"
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
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
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
	return m, nil
}

func (m *Migrator) ensureVersionTable() error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS ` + migrationsTable + ` (version INTEGER PRIMARY KEY, dirty INTEGER NOT NULL DEFAULT 0);
		CREATE UNIQUE INDEX IF NOT EXISTS version_unique ON ` + migrationsTable + `(version);
	`); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (m *Migrator) Up(steps int) (uint, error) {
	version, err := m.up(steps)
	if errors.Is(err, errNoChange) {
		return version, nil
	}
	if err != nil {
		return 0, fmt.Errorf("migration up failed: %w", err)
	}
	return version, nil
}

func (m *Migrator) Down(steps int) (uint, error) {
	version, err := m.down(steps)
	if errors.Is(err, errNoChange) {
		return version, nil
	}
	if err != nil {
		return 0, fmt.Errorf("migration down failed: %w", err)
	}
	return version, nil
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

func (m *Migrator) DB() *sql.DB {
	return m.db
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
