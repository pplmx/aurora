package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/pplmx/aurora/internal/infra/sqlite"
)

func (s *BackupService) Verify(ctx context.Context, backupPath string) error {
	_, err := s.verifiedArchiveMetadata(ctx, backupPath)
	return err
}

// verifiedArchiveMetadata returns the exact metadata used for a successful
// Verify so Restore can bind its copy to the already-observed database hashes
// instead of re-reading mutable archive metadata.
func (s *BackupService) verifiedArchiveMetadata(ctx context.Context, backupPath string) (*BackupMetadata, error) {
	if err := s.recoverInterruptedCreate(ctx, backupPath); err != nil {
		return nil, fmt.Errorf("recover interrupted backup create: %w", err)
	}
	metadata, err := verifyArchiveContents(backupPath)
	if err != nil {
		return nil, err
	}
	if err := s.validateArchiveDatabaseSet(metadata.Databases); err != nil {
		return nil, err
	}
	return metadata, nil
}

// verifyArchiveContents checks whether a backup is internally complete. It
// deliberately excludes the installation-specific database-set check so crash
// recovery can distinguish an interrupted promotion from a valid archive that
// simply belongs to a different configuration.
func verifyArchiveContents(backupPath string) (*BackupMetadata, error) {
	metaPath := filepath.Join(backupPath, "metadata.json")
	metaInfo, err := os.Lstat(metaPath)
	if err != nil {
		return nil, fmt.Errorf("metadata not found: %w", err)
	}
	if !metaInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("metadata path %q is not a regular file", metaPath)
	}
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("metadata not found: %w", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(metaData, &metadata); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	storedChecksum := metadata.Checksum
	metadata.Checksum = ""
	recomputed, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata for verification: %w", err)
	}
	computed := fmt.Sprintf("%x", sha256.Sum256(recomputed))
	if storedChecksum != computed {
		return nil, fmt.Errorf("checksum mismatch: backup may be corrupted")
	}

	for _, name := range metadata.Databases {
		if err := validateDatabaseName(name); err != nil {
			return nil, fmt.Errorf("invalid database name in backup metadata: %w", err)
		}
	}
	if err := validateUniqueArchiveDatabaseNames(metadata.Databases); err != nil {
		return nil, err
	}
	if metadata.DatabaseChecksums == nil {
		metadata.DatabaseChecksums = make(map[string]string, len(metadata.Databases))
	}

	for _, name := range metadata.Databases {
		dbPath := filepath.Join(backupPath, name+".db")
		dbInfo, err := os.Lstat(dbPath)
		if err != nil {
			return nil, fmt.Errorf("missing database file: %s", name)
		}
		if !dbInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("database file %q is not a regular file", dbPath)
		}

		got, err := verifyDatabaseFile(dbPath, name, metadata.DatabaseChecksums[name])
		if err != nil {
			return nil, err
		}
		// Keep an observed hash even for legacy metadata without per-database
		// checksums. Restore compares the bytes it copied against this value.
		metadata.DatabaseChecksums[name] = got
	}

	return &metadata, nil
}

// verifyDatabaseFile enforces the stored content hash when one exists, always
// checks that the file is a readable SQLite database, and returns the observed
// hash so Restore can detect a file changing after verification. Legacy
// backups without a stored hash still retain the table-presence contract.
func verifyDatabaseFile(path, name, wantChecksum string) (string, error) {
	got, err := fileSHA256(path)
	if err != nil {
		return "", fmt.Errorf("checksum %s: %w", name, err)
	}
	if wantChecksum != "" {
		if got != wantChecksum {
			return "", fmt.Errorf("database %s checksum mismatch: backup may be corrupted", name)
		}
	}

	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		return "", fmt.Errorf("cannot open %s: %w", name, err)
	}
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil || count == 0 {
		_ = db.Close()
		return "", fmt.Errorf("invalid database: %s", name)
	}
	if err := db.Close(); err != nil {
		return "", fmt.Errorf("close %s: %w", name, err)
	}
	return got, nil
}
