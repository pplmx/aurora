package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type Migrator struct {
	db      *sql.DB
	m       *migrate.Migrate
	dbPath  string
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

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	migrator := &Migrator{
		db:      db,
		dbPath:  dbPath,
		migPath: migPath,
	}

	if err := migrator.initMigrate(); err != nil {
		db.Close()
		return nil, err
	}

	return migrator, nil
}

func (m *Migrator) initMigrate() error {
	instance, err := sqlite3.WithInstance(m.db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite3 driver: %w", err)
	}

	absMigPath, err := filepath.Abs(m.migPath)
	if err != nil {
		absMigPath = m.migPath
	}

	src, err := (&file.File{}).Open("file://" + absMigPath)
	if err != nil {
		return fmt.Errorf("failed to open migration source: %w", err)
	}

	m.m, err = migrate.NewWithInstance("file", src, "aurora", instance)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	return nil
}

func (m *Migrator) Up(steps int) (uint, error) {
	if steps <= 0 {
		if err := m.m.Up(); err != nil && err != migrate.ErrNoChange {
			return 0, fmt.Errorf("migration up failed: %w", err)
		}
	} else {
		if err := m.m.Steps(steps); err != nil && err != migrate.ErrNoChange {
			return 0, fmt.Errorf("migration steps failed: %w", err)
		}
	}

	version, _, err := m.m.Version()
	if err != nil {
		return 0, err
	}

	return version, nil
}

func (m *Migrator) Down(steps int) (uint, error) {
	if steps <= 0 {
		if err := m.m.Down(); err != nil && err != migrate.ErrNoChange {
			return 0, fmt.Errorf("migration down failed: %w", err)
		}
	} else {
		if err := m.m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
			return 0, fmt.Errorf("migration steps failed: %w", err)
		}
	}

	version, _, err := m.m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, err
	}

	return version, nil
}

func (m *Migrator) Status() (*MigrationStatus, error) {
	version, dirty, err := m.m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return nil, err
	}

	pendingVersions, err := m.getPendingMigrations()
	if err != nil {
		return nil, err
	}

	return &MigrationStatus{
		Current:         version,
		Dirty:           dirty,
		Applied:         applied,
		PendingVersions: pendingVersions,
	}, nil
}

func (m *Migrator) getAppliedMigrations() ([]uint, error) {
	currentVersion, dirty, err := m.m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	if currentVersion == 0 && !dirty {
		return []uint{}, nil
	}

	var versions []uint
	for v := uint(1); v <= currentVersion; v++ {
		versions = append(versions, v)
	}

	return versions, nil
}

func (m *Migrator) getPendingMigrations() ([]uint, error) {
	currentVersion, _, err := m.m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	entries, err := os.ReadDir(m.migPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	seen := make(map[uint]bool)
	var versions []uint
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		var v uint
		_, err := fmt.Sscanf(name, "%d", &v)
		if err != nil {
			continue
		}
		if v > currentVersion && !seen[v] {
			versions = append(versions, v)
			seen[v] = true
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i] < versions[j]
	})

	return versions, nil
}

func (m *Migrator) Version() (uint, error) {
	version, _, err := m.m.Version()
	if err == migrate.ErrNilVersion {
		return 0, nil
	}
	return version, err
}

func (m *Migrator) Close() error {
	if m.m != nil {
		_, _ = m.m.Close()
	}
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
	defer m.Close()

	if err := m.m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("auto migration failed: %w", err)
	}

	return nil
}
