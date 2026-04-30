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
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}

		if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			db.Close()
			return nil, fmt.Errorf("checkpoint %s: %w", name, err)
		}
		db.Close()

		src, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open source %s: %w", name, err)
		}
		destPath := filepath.Join(output, name+".db")
		dest, err := os.Create(destPath)
		if err != nil {
			src.Close()
			return nil, fmt.Errorf("create dest %s: %w", name, err)
		}
		if _, err := io.Copy(dest, src); err != nil {
			src.Close()
			dest.Close()
			return nil, fmt.Errorf("copy %s: %w", name, err)
		}
		src.Close()
		dest.Close()
	}

	metadata := BackupMetadata{
		Version:       "1.2",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Databases:     make([]string, 0, len(s.dbPaths)),
		SchemaVersion: schemaVersion,
	}
	for name := range s.dbPaths {
		metadata.Databases = append(metadata.Databases, name)
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
	filepath.Walk(output, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return &BackupResult{
		File:          output,
		Size:          totalSize,
		Checksum:      checksum,
		SchemaVersion: schemaVersion,
	}, nil
}

func (s *BackupService) getSchemaVersion() uint {
	for _, path := range s.dbPaths {
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			continue
		}
		defer db.Close()

		var version int
		err = db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
		if err == nil {
			return uint(version)
		}
	}
	return 0
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
	recomputed, _ := json.Marshal(metadata)
	computed := fmt.Sprintf("%x", sha256.Sum256(recomputed))
	if storedChecksum != computed {
		return fmt.Errorf("checksum mismatch: backup may be corrupted")
	}

	for _, name := range metadata.Databases {
		dbPath := filepath.Join(backupPath, name+".db")
		if _, err := os.Stat(dbPath); err != nil {
			return fmt.Errorf("missing database file: %s", name)
		}

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", name, err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
		if err != nil || count == 0 {
			db.Close()
			return fmt.Errorf("invalid database: %s", name)
		}
		db.Close()
	}

	return nil
}

func (s *BackupService) Restore(ctx context.Context, backupPath string) error {
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
			dest, err := os.Create(prePath)
			if err != nil {
				src.Close()
				return fmt.Errorf("create pre-restore %s: %w", name, err)
			}
			if _, err := io.Copy(dest, src); err != nil {
				src.Close()
				dest.Close()
				return fmt.Errorf("backup current %s: %w", name, err)
			}
			src.Close()
			dest.Close()
		}

		backupDbPath := filepath.Join(backupPath, name+".db")
		src, err := os.Open(backupDbPath)
		if err != nil {
			return fmt.Errorf("open backup %s: %w", name, err)
		}
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			src.Close()
			return fmt.Errorf("remove dest %s: %w", name, err)
		}
		dest, err := os.Create(destPath)
		if err != nil {
			src.Close()
			return fmt.Errorf("create dest %s: %w", name, err)
		}
		if _, err := io.Copy(dest, src); err != nil {
			src.Close()
			dest.Close()
			return fmt.Errorf("restore %s: %w", name, err)
		}
		src.Close()
		dest.Close()
	}

	return nil
}
