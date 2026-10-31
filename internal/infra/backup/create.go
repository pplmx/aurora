// Create's two-phase write: snapshot every database to a temp sibling, promote
// every snapshot at once, then write metadata. The helpers that recover a run
// interrupted by SIGKILL or power loss live in create_recovery.go, because that
// code runs before a Create rather than during one, and reading it beside the
// happy path hid which of the two a function belonged to.
package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	_ "modernc.org/sqlite"

	"github.com/pplmx/aurora/internal/infra/sqlite"
)

// snapshot tracks one DB's temp sibling inside a Create run so an aborted run
// can clean all of them up before returning (see Create's two-phase design).
type snapshot struct {
	name         string
	destPath     string
	tmpPath      string
	previousPath string
	hadDest      bool
}

// removeTemps cleans up the not-yet-promoted snapshot temps of an aborted
// Create so a failed re-run never leaves half-written .tmp files behind
// (each temp is named <name>.db.tmp and would be removed at the top of the
// next Create anyway).
func (s *BackupService) removeTemps(snaps []snapshot) {
	for _, snap := range snaps {
		_ = os.Remove(snap.tmpPath)
	}
}

func (s *BackupService) Create(ctx context.Context, output string) (result *BackupResult, err error) {
	if s.configErr != nil {
		return nil, fmt.Errorf("invalid backup database configuration: %w", s.configErr)
	}
	if err := mkdirAllDurable(output, 0o700); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	if err := s.recoverInterruptedCreate(ctx, output); err != nil {
		return nil, fmt.Errorf("recover interrupted backup create: %w", err)
	}

	schemaVersion := s.getSchemaVersion()

	// Two-phase (snapshot-all → promote-all) so a backing up into an existing
	// output directory is all-or-nothing. The old code removed destPath before
	// VACUUM INTO, so a mid-run failure (disk full, I/O error, SIGKILL) left the
	// previous good snapshot of that DB deleted while metadata.json still
	// described it — the backup set became unverifiable/unrestorable and the
	// operator's only good copy was gone (TASK-109, ISS-101). Now every DB is
	// first snapshotted to a temp sibling (failure → remove temps, previous
	// .db set untouched), then all temps are atomically renamed into place,
	// then metadata is written. A rollback copy of the previous visible set is
	// taken before promotion, so a failure in the promotion or metadata tail
	// restores that set instead of leaving new DBs under old metadata.
	snapshots := make([]snapshot, 0, len(s.dbPaths))
	names := make([]string, 0, len(s.dbPaths))
	for name := range s.dbPaths {
		if err := validateDatabaseName(name); err != nil {
			return nil, fmt.Errorf("invalid configured database name: %w", err)
		}
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		path := s.dbPaths[name]
		destPath := filepath.Join(output, name+".db")
		tmpPath := destPath + ".tmp"

		srcInfo, statErr := os.Stat(path)
		if statErr != nil {
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("stat source %s: %w", name, statErr)
		}
		if destInfo, statErr := os.Lstat(destPath); statErr == nil {
			if destInfo.IsDir() {
				s.removeTemps(snapshots)
				return nil, fmt.Errorf("destination %s path %q is a directory: refusing to overwrite it", name, destPath)
			}
			if !destInfo.Mode().IsRegular() {
				s.removeTemps(snapshots)
				return nil, fmt.Errorf("destination %s path %q is not a regular file: refusing to overwrite it", name, destPath)
			}
			// Refuse to write the backup file over the live source (ISS-071):
			// `aurora backup create ./data` with the DB at ./data/aurora.db
			// sets destPath == path — VACUUM INTO would otherwise clobber the
			// live database. os.SameFile catches the aliasing regardless of
			// ./data vs data vs symlinked paths.
			if os.SameFile(srcInfo, destInfo) {
				s.removeTemps(snapshots)
				return nil, fmt.Errorf("refusing to back up %s into itself: output directory %q holds the live database %q", name, output, path)
			}
		} else if !os.IsNotExist(statErr) {
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("stat destination %s: %w", name, statErr)
		}
		// A stale temp can survive a crash right before the rename below;
		// VACUUM INTO requires its destination to not exist.
		if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("remove stale temp %s: %w", name, err)
		}

		// VACUUM INTO is an online-backup statement: it writes a complete,
		// self-contained snapshot (committed WAL frames included) to a fresh
		// file even while the server keeps writing to the source — unlike the
		// old PRAGMA wal_checkpoint(TRUNCATE) (whose busy/partial result code
		// was ignored) + bare .db file copy, which silently produced a stale
		// snapshot missing whatever committed between checkpoint and copy
		// (v1.75, ISS-082).
		db, err := sql.Open("sqlite", sqlite.DSN(path))
		if err != nil {
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("open %s: %w", name, err)
		}
		if _, err := db.Exec("VACUUM INTO " + sqlLiteral(tmpPath)); err != nil {
			_ = db.Close()
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("snapshot %s: %w", name, err)
		}
		if err := db.Close(); err != nil {
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("close %s: %w", name, err)
		}
		snapshots = append(snapshots, snapshot{name: name, destPath: destPath, tmpPath: tmpPath})
		if err := restrictBackupFile(tmpPath); err != nil {
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("restrict snapshot %s: %w", name, err)
		}
		if err := syncFileAtPath(tmpPath); err != nil {
			s.removeTemps(snapshots)
			return nil, fmt.Errorf("sync snapshot %s: %w", name, err)
		}
	}

	rollback, err := stageCreateRollback(output, snapshots)
	if err != nil {
		s.removeTemps(snapshots)
		return nil, err
	}
	promoted := make([]int, 0, len(snapshots))
	defer func() {
		if err == nil {
			_ = os.RemoveAll(rollback.dir)
			return
		}
		if rollbackErr := rollbackCreate(snapshots, promoted, rollback); rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf(
				"rollback backup set (previous files retained in %q): %w",
				rollback.dir,
				rollbackErr,
			))
			return
		}
		_ = os.RemoveAll(rollback.dir)
	}()

	// Phase 2: promote every completed snapshot into place. os.Rename is
	// atomic on the same filesystem, so each prior .db is only replaced once
	// its fully-written replacement is ready.
	rename := s.renameFn
	if rename == nil {
		rename = os.Rename
	}
	for i, snap := range snapshots {
		if err := rename(snap.tmpPath, snap.destPath); err != nil {
			return nil, fmt.Errorf("promote snapshot %s: %w", snap.name, err)
		}
		promoted = append(promoted, i)
	}

	metadata := BackupMetadata{
		Version:           "1.2",
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Databases:         make([]string, 0, len(s.dbPaths)),
		SchemaVersion:     schemaVersion,
		DatabaseChecksums: make(map[string]string, len(s.dbPaths)),
	}
	for _, snap := range snapshots {
		metadata.Databases = append(metadata.Databases, snap.name)
		sum, err := fileSHA256(snap.destPath)
		if err != nil {
			return nil, fmt.Errorf("checksum backup %s: %w", snap.name, err)
		}
		metadata.DatabaseChecksums[snap.name] = sum
	}

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
	// Write metadata atomically (TASK-120): a plain os.WriteFile over the live
	// metadata.json can leave a truncated file on top of brand-new .db files if
	// the process dies mid-write or the write short-fails — the backup set then
	// fails Verify with no good copy recoverable. Write to the private
	// same-filesystem rollback directory, then os.Rename into place.
	writeFile := s.writeFileFn
	if writeFile == nil {
		writeFile = os.WriteFile
	}
	if err := writeFile(rollback.metaTmpPath, metaData, 0o600); err != nil {
		return nil, fmt.Errorf("write metadata: %w", err)
	}
	if err := restrictBackupFile(rollback.metaTmpPath); err != nil {
		return nil, fmt.Errorf("restrict metadata: %w", err)
	}
	if err := syncFileAtPath(rollback.metaTmpPath); err != nil {
		return nil, fmt.Errorf("sync metadata: %w", err)
	}
	if err := os.Rename(rollback.metaTmpPath, rollback.metaPath); err != nil {
		return nil, fmt.Errorf("promote metadata: %w", err)
	}
	if err := syncDirFn(output); err != nil {
		return nil, fmt.Errorf("sync output directory: %w", err)
	}

	totalSize := int64(0)
	walkErr := filepath.Walk(output, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == rollback.dir && info.IsDir() {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk backup output: %w", walkErr)
	}

	return &BackupResult{
		Dir:           output,
		Size:          totalSize,
		Checksum:      checksum,
		SchemaVersion: schemaVersion,
	}, nil
}
