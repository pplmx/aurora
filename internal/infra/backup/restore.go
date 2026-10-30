package backup

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pplmx/aurora/internal/utils"
)

func (s *BackupService) Restore(ctx context.Context, backupPath string) error {
	if s.configErr != nil {
		return fmt.Errorf("invalid backup database configuration: %w", s.configErr)
	}
	// A shell-completed directory path often ends in a separator. Normalize it
	// before deriving the sibling .pre_restore directory so the safety copy
	// never moves inside the archive being restored.
	backupPath = filepath.Clean(backupPath)

	// Safety guard (v1.20): never clobber the live databases with a backup
	// that fails integrity verification. Previously a corrupt/truncated backup
	// was copied over a good DB, with recovery possible only through the
	// hidden .pre_restore staging dir. Verifying first means a bad backup is
	// rejected before any live file is touched.
	metadata, err := s.verifiedArchiveMetadata(ctx, backupPath)
	if err != nil {
		return fmt.Errorf("refusing to restore: %w", err)
	}

	operations, err := s.planRestore(backupPath, metadata)
	if err != nil {
		return err
	}

	preRestoreDir := backupPath + ".pre_restore"
	if err := mkdirAllDurable(preRestoreDir, 0o700); err != nil {
		return fmt.Errorf("create pre-restore dir: %w", err)
	}
	if err := restrictBackupDir(preRestoreDir); err != nil {
		return fmt.Errorf("restrict pre-restore dir: %w", err)
	}

	if err := prepareRestore(operations, preRestoreDir); err != nil {
		cleanupRestoreTemps(operations)
		return err
	}
	if err := writePreRestoreMetadata(preRestoreDir, operations); err != nil {
		cleanupRestoreTemps(operations)
		return err
	}

	promoted, err := s.promoteRestore(ctx, operations)
	if err != nil {
		rollbackErr := rollbackRestore(operations, promoted)
		cleanupRestoreTemps(operations)
		if rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("rollback restored databases: %w", rollbackErr))
		}
		return fmt.Errorf("restore failed and was rolled back: %w", err)
	}

	cleanupRestoreTemps(operations)
	return nil
}

type restoreOperation struct {
	name       string
	backupPath string
	destPath   string
	prePath    string
	tmpPath    string
	hadDest    bool
	// expectedChecksum comes from the verified in-memory metadata. It is never
	// re-read from the archive, so a metadata swap cannot bless changed bytes.
	expectedChecksum string
}

// planRestore validates every database before any live file is read or
// replaced. The archive and the running configuration must describe the same
// set: silently restoring a subset would leave a multi-store installation with
// mixed generations even though Restore returned success.
func (s *BackupService) planRestore(backupDir string, metadata *BackupMetadata) ([]restoreOperation, error) {
	if err := s.validateArchiveDatabaseSet(metadata.Databases); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(s.dbPaths))
	for name := range s.dbPaths {
		names = append(names, name)
	}
	slices.Sort(names)

	operations := make([]restoreOperation, 0, len(names))
	for _, name := range names {
		destPath, err := utils.ResolveFinalSymlink(s.dbPaths[name])
		if err != nil {
			return nil, fmt.Errorf("resolve restore destination %s: %w", name, err)
		}
		backupDBPath := filepath.Join(backupDir, name+".db")
		backupInfo, err := os.Lstat(backupDBPath)
		if err != nil {
			return nil, fmt.Errorf("stat backup %s: %w", name, err)
		}
		if !backupInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("backup database %q is not a regular file", backupDBPath)
		}

		op := restoreOperation{
			name:             name,
			backupPath:       backupDBPath,
			destPath:         destPath,
			tmpPath:          destPath + ".tmp",
			expectedChecksum: metadata.DatabaseChecksums[name],
		}
		destInfo, err := os.Stat(destPath)
		switch {
		case err == nil:
			if destInfo.IsDir() {
				return nil, fmt.Errorf("restore destination %q is a directory", destPath)
			}
			if os.SameFile(backupInfo, destInfo) {
				return nil, fmt.Errorf("refusing to restore %s: backup database %q aliases the live database %q", name, backupDBPath, destPath)
			}
			op.hadDest = true
		case !os.IsNotExist(err):
			return nil, fmt.Errorf("stat restore destination %s: %w", name, err)
		}
		operations = append(operations, op)
	}
	return operations, nil
}

// prepareRestore makes every safety snapshot and replacement file before the
// first destination is renamed. Any preparation failure therefore leaves all
// live databases untouched.
// writePreRestoreMetadata turns the pre-restore copies into a verifiable
// archive. Without metadata.json the undo copy was a bare database that Verify
// and Restore refused, so an operator who restored the wrong backup could only
// copy files back by hand — the one documented recovery path that the safety
// copy did not actually support. A first-time restore, with no previous
// database, writes nothing.
func writePreRestoreMetadata(dir string, operations []restoreOperation) error {
	metadata := BackupMetadata{
		Version:           "1.2",
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		DatabaseChecksums: make(map[string]string, len(operations)),
	}
	for i := range operations {
		op := &operations[i]
		if !op.hadDest {
			continue
		}
		sum, err := fileSHA256(op.prePath)
		if err != nil {
			return fmt.Errorf("checksum pre-restore %s: %w", op.name, err)
		}
		metadata.Databases = append(metadata.Databases, op.name)
		metadata.DatabaseChecksums[op.name] = sum
		if v := querySchemaVersion(op.prePath); v > metadata.SchemaVersion {
			metadata.SchemaVersion = v
		}
	}
	if len(metadata.Databases) == 0 {
		return nil
	}
	checksumData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal pre-restore metadata: %w", err)
	}
	metadata.Checksum = fmt.Sprintf("%x", sha256.Sum256(checksumData))
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pre-restore metadata: %w", err)
	}
	path := filepath.Join(dir, "metadata.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write pre-restore metadata: %w", err)
	}
	if err := restrictBackupFile(path); err != nil {
		return fmt.Errorf("restrict pre-restore metadata: %w", err)
	}
	return syncFileAtPath(path)
}

func prepareRestore(operations []restoreOperation, preRestoreDir string) error {
	for i := range operations {
		op := &operations[i]
		if op.hadDest {
			op.prePath = filepath.Join(preRestoreDir, op.name+".db")
			if err := os.Remove(op.prePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove stale pre-restore %s: %w", op.name, err)
			}
			// The safety copy must include committed WAL frames so rollback
			// cannot resurrect an older database state (TASK-116, ISS-108).
			if err := vacuumInto(op.destPath, op.prePath); err != nil {
				_ = os.Remove(op.prePath)
				return fmt.Errorf("snapshot current %s before restore: %w", op.name, err)
			}
			if err := restrictBackupFile(op.prePath); err != nil {
				_ = os.Remove(op.prePath)
				return fmt.Errorf("restrict pre-restore %s: %w", op.name, err)
			}
			if err := syncFileAtPath(op.prePath); err != nil {
				_ = os.Remove(op.prePath)
				return fmt.Errorf("sync pre-restore %s: %w", op.name, err)
			}
		}
		gotChecksum, err := copyFileAtomicWithHash(op.backupPath, op.tmpPath)
		if err != nil {
			return fmt.Errorf("prepare restored %s: %w", op.name, err)
		}
		if op.expectedChecksum == "" || gotChecksum != op.expectedChecksum {
			_ = os.Remove(op.tmpPath)
			return fmt.Errorf(
				"prepare restored %s: checksum mismatch after verification: archive changed during restore",
				op.name,
			)
		}
	}
	if err := syncDirFn(preRestoreDir); err != nil {
		return fmt.Errorf("sync pre-restore directory: %w", err)
	}
	return nil
}

func (s *BackupService) promoteRestore(ctx context.Context, operations []restoreOperation) ([]int, error) {
	promoted := make([]int, 0, len(operations))
	for i := range operations {
		op := &operations[i]
		if err := ctx.Err(); err != nil {
			return promoted, fmt.Errorf("restore cancelled before %s: %w", op.name, err)
		}
		if err := os.Rename(op.tmpPath, op.destPath); err != nil {
			return promoted, fmt.Errorf("promote restored %s: %w", op.name, err)
		}
		promoted = append(promoted, i)

		// A stale WAL/SHM or rollback journal from the pre-restore DB could be
		// replayed against the new archive on next open (v1.75, ISS-082).
		if err := s.removeSidecars(op.destPath, op.name); err != nil {
			return promoted, err
		}
		if err := syncDirFn(filepath.Dir(op.destPath)); err != nil {
			return promoted, fmt.Errorf("sync restored %s: %w", op.name, err)
		}
	}
	return promoted, nil
}

func (s *BackupService) removeSidecars(destPath, name string) error {
	remove := s.removeSidecarFn
	if remove == nil {
		remove = os.Remove
	}
	for _, sidecar := range utils.SQLiteSidecarPaths(destPath) {
		if err := remove(sidecar); err != nil && !os.IsNotExist(err) {
			suffix := strings.TrimPrefix(sidecar, destPath)
			return fmt.Errorf("remove stale %s for %s: %w", suffix, name, err)
		}
	}
	return nil
}

func rollbackRestore(operations []restoreOperation, promoted []int) error {
	var rollbackErrors []error
	for i := len(promoted) - 1; i >= 0; i-- {
		op := operations[promoted[i]]
		if op.hadDest {
			if err := copyFileAtomic(op.prePath, op.tmpPath); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("recover %s: %w", op.name, err))
				continue
			}
			if err := os.Rename(op.tmpPath, op.destPath); err != nil {
				_ = os.Remove(op.tmpPath)
				rollbackErrors = append(rollbackErrors, fmt.Errorf("recover %s: %w", op.name, err))
				continue
			}
		} else if err := os.Remove(op.destPath); err != nil && !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("remove newly created %s: %w", op.name, err))
			continue
		}

		for _, sidecar := range utils.SQLiteSidecarPaths(op.destPath) {
			if err := os.Remove(sidecar); err != nil && !os.IsNotExist(err) {
				suffix := strings.TrimPrefix(sidecar, op.destPath)
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove rolled-back %s%s: %w", op.name, suffix, err))
			}
		}
		if err := syncDirFn(filepath.Dir(op.destPath)); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("sync rolled-back %s: %w", op.name, err))
		}
	}
	return errors.Join(rollbackErrors...)
}

func cleanupRestoreTemps(operations []restoreOperation) {
	for _, op := range operations {
		_ = os.Remove(op.tmpPath)
	}
}
