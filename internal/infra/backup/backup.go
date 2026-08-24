package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type BackupService struct {
	dbPaths map[string]string
}

type BackupResult struct {
	File          string
	Size          int64
	Checksum      string
	SchemaVersion uint
}

type BackupMetadata struct {
	Version       string   `json:"version"`
	Timestamp     string   `json:"timestamp"`
	Checksum      string   `json:"checksum"`
	Databases     []string `json:"databases"`
	SchemaVersion uint     `json:"schema_version"`
	// DatabaseChecksums maps each backed-up database name to the SHA-256 of
	// its file contents at backup time (v1.55). Previously Verify hashed only
	// the metadata JSON, so a truncated or bit-corrupted .db that still opened
	// with at least one table passed as valid and could be restored over a
	// good database. Recording the content hash lets Verify/Restore guarantee
	// the actual data is intact, not just the metadata. The field is
	// `omitempty` so backups created by older versions (which never wrote it)
	// remain verifiable via the existing table-presence check.
	DatabaseChecksums map[string]string `json:"database_checksums,omitempty"`
}

// fileSHA256 returns the hex SHA-256 of a file's contents.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// sqlLiteral renders s as a single-quoted SQL string literal, doubling any
// embedded single quotes and rejecting NUL bytes (which SQLite forbids).
// Used for file paths in VACUUM INTO, which does not accept placeholders.
func sqlLiteral(s string) string {
	if strings.ContainsRune(s, 0) {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func NewBackupService(dbPaths map[string]string) *BackupService {
	return &BackupService{dbPaths: dbPaths}
}

func (s *BackupService) Create(ctx context.Context, output string) (*BackupResult, error) {
	if err := os.MkdirAll(output, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	schemaVersion := s.getSchemaVersion()

	for name, path := range s.dbPaths {
		destPath := filepath.Join(output, name+".db")

		srcInfo, statErr := os.Stat(path)
		if statErr != nil {
			return nil, fmt.Errorf("stat source %s: %w", name, statErr)
		}
		if destInfo, statErr := os.Stat(destPath); statErr == nil {
			if destInfo.IsDir() {
				return nil, fmt.Errorf("destination %s path %q is a directory: refusing to overwrite it", name, destPath)
			}
			// Refuse to write the backup file over the live source (ISS-071):
			// `aurora backup create ./data` with the DB at ./data/aurora.db
			// sets destPath == path — VACUUM INTO would otherwise clobber the
			// live database. os.SameFile catches the aliasing regardless of
			// ./data vs data vs symlinked paths.
			if os.SameFile(srcInfo, destInfo) {
				return nil, fmt.Errorf("refusing to back up %s into itself: output directory %q holds the live database %q", name, output, path)
			}
		}
		// VACUUM INTO requires the destination to not exist.
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("remove dest %s: %w", name, err)
		}

		// VACUUM INTO is an online-backup statement: it writes a complete,
		// self-contained snapshot (committed WAL frames included) to a fresh
		// file even while the server keeps writing to the source — unlike the
		// old PRAGMA wal_checkpoint(TRUNCATE) (whose busy/partial result code
		// was ignored) + bare .db file copy, which silently produced a stale
		// snapshot missing whatever committed between checkpoint and copy
		// (v1.75, ISS-082).
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}
		if _, err := db.Exec("VACUUM INTO " + sqlLiteral(destPath)); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("snapshot %s: %w", name, err)
		}
		if err := db.Close(); err != nil {
			return nil, fmt.Errorf("close %s: %w", name, err)
		}
	}

	metadata := BackupMetadata{
		Version:           "1.2",
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Databases:         make([]string, 0, len(s.dbPaths)),
		SchemaVersion:     schemaVersion,
		DatabaseChecksums: make(map[string]string, len(s.dbPaths)),
	}
	for name := range s.dbPaths {
		metadata.Databases = append(metadata.Databases, name)
		sum, err := fileSHA256(filepath.Join(output, name+".db"))
		if err != nil {
			return nil, fmt.Errorf("checksum backup %s: %w", name, err)
		}
		metadata.DatabaseChecksums[name] = sum
	}

	metaPath := filepath.Join(output, "metadata.json")
	checksumData, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(checksumData))
	metadata.Checksum = checksum
	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0640); err != nil {
		return nil, fmt.Errorf("write metadata: %w", err)
	}

	totalSize := int64(0)
	walkErr := filepath.Walk(output, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk backup output: %w", walkErr)
	}

	return &BackupResult{
		File:          output,
		Size:          totalSize,
		Checksum:      checksum,
		SchemaVersion: schemaVersion,
	}, nil
}

// getSchemaVersion returns the highest schema_migrations.version found
// across all configured DBs. Returns 0 if none of the DBs have a
// schema_migrations table or all queries fail.
func (s *BackupService) getSchemaVersion() uint {
	var highest uint
	for _, path := range s.dbPaths {
		v := querySchemaVersion(path)
		if v > highest {
			highest = v
		}
	}
	return highest
}

// querySchemaVersion reads schema_migrations.version from a single SQLite DB.
// Returns 0 if the table is missing or the file can't be opened; in either
// case we treat that DB as "version 0" and let the caller decide the overall
// result (typically the highest across all DBs wins).
func querySchemaVersion(path string) uint {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return 0
	}
	defer func() { _ = db.Close() }()

	var version int
	if err := db.QueryRow(
		"SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1",
	).Scan(&version); err != nil {
		return 0
	}
	if version < 0 {
		return 0
	}
	return uint(version)
}

func (s *BackupService) Verify(ctx context.Context, backupPath string) error {
	metaPath := filepath.Join(backupPath, "metadata.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("metadata not found: %w", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(metaData, &metadata); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	storedChecksum := metadata.Checksum
	metadata.Checksum = ""
	recomputed, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata for verification: %w", err)
	}
	computed := fmt.Sprintf("%x", sha256.Sum256(recomputed))
	if storedChecksum != computed {
		return fmt.Errorf("checksum mismatch: backup may be corrupted")
	}

	for _, name := range metadata.Databases {
		dbPath := filepath.Join(backupPath, name+".db")
		if _, err := os.Stat(dbPath); err != nil {
			return fmt.Errorf("missing database file: %s", name)
		}

		// Content integrity (v1.55): when the backup recorded a per-database
		// checksum, verify the file's actual bytes match. This catches a
		// truncated or bit-corrupted .db that still opens with >=1 table — the
		// gap the metadata-only checksum left open. Backups without the field
		// (created by older versions) fall through to the table-presence check
		// below so existing archives remain verifiable.
		if want, ok := metadata.DatabaseChecksums[name]; ok && want != "" {
			got, err := fileSHA256(dbPath)
			if err != nil {
				return fmt.Errorf("checksum %s: %w", name, err)
			}
			if got != want {
				return fmt.Errorf("database %s checksum mismatch: backup may be corrupted", name)
			}
		} else {
			db, err := sql.Open("sqlite3", dbPath)
			if err != nil {
				return fmt.Errorf("cannot open %s: %w", name, err)
			}

			var count int
			err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
			if err != nil || count == 0 {
				_ = db.Close()
				return fmt.Errorf("invalid database: %s", name)
			}
			if err := db.Close(); err != nil {
				return fmt.Errorf("close %s: %w", name, err)
			}
		}
	}

	return nil
}

func (s *BackupService) Restore(ctx context.Context, backupPath string) error {
	// Safety guard (v1.20): never clobber the live databases with a backup
	// that fails integrity verification. Previously a corrupt/truncated backup
	// was copied over a good DB, with recovery possible only through the
	// hidden .pre_restore staging dir. Verifying first means a bad backup is
	// rejected before any live file is touched.
	if err := s.Verify(ctx, backupPath); err != nil {
		return fmt.Errorf("refusing to restore: %w", err)
	}

	metaPath := filepath.Join(backupPath, "metadata.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("read metadata: %w", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(metaData, &metadata); err != nil {
		return fmt.Errorf("parse metadata: %w", err)
	}

	preRestoreDir := backupPath + ".pre_restore"
	if err := os.MkdirAll(preRestoreDir, 0755); err != nil {
		return fmt.Errorf("create pre-restore dir: %w", err)
	}

	for name, destPath := range s.dbPaths {
		if _, err := os.Stat(destPath); err == nil {
			src, err := os.Open(destPath)
			if err != nil {
				return fmt.Errorf("open current %s: %w", name, err)
			}
			prePath := filepath.Join(preRestoreDir, name+".db")
			dest, err := os.OpenFile(prePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
			if err != nil {
				_ = src.Close()
				return fmt.Errorf("create pre-restore %s: %w", name, err)
			}
			if _, err := io.Copy(dest, src); err != nil {
				_ = src.Close()
				_ = dest.Close()
				return fmt.Errorf("backup current %s: %w", name, err)
			}
			if err := src.Close(); err != nil {
				_ = dest.Close()
				return fmt.Errorf("close current %s: %w", name, err)
			}
			if err := dest.Close(); err != nil {
				return fmt.Errorf("close pre-restore %s: %w", name, err)
			}
		}

		backupDbPath := filepath.Join(backupPath, name+".db")
		src, err := os.Open(backupDbPath)
		if err != nil {
			return fmt.Errorf("open backup %s: %w", name, err)
		}
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			_ = src.Close()
			return fmt.Errorf("remove dest %s: %w", name, err)
		}
		dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			_ = src.Close()
			return fmt.Errorf("create dest %s: %w", name, err)
		}
		if _, err := io.Copy(dest, src); err != nil {
			_ = src.Close()
			_ = dest.Close()
			return fmt.Errorf("restore %s: %w", name, err)
		}
		if err := src.Close(); err != nil {
			_ = dest.Close()
			return fmt.Errorf("close backup %s: %w", name, err)
		}
		if err := dest.Close(); err != nil {
			return fmt.Errorf("close dest %s: %w", name, err)
		}

		// The restored archive is a complete standalone snapshot; a stale
		// -wal/-shm left next to it from the pre-restore live DB would be
		// replayed against the mismatched .db on next open, resurrecting
		// pre-restore committed frames (v1.75, ISS-082).
		for _, ext := range []string{"-wal", "-shm"} {
			if err := os.Remove(destPath + ext); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove stale %s for %s: %w", ext, name, err)
			}
		}
	}

	return nil
}
