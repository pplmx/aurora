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
	"strings"
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

const (
	createRollbackPrefix  = ".create-rollback-"
	createRollbackPattern = createRollbackPrefix + "*"
)

type createRollbackCandidate struct {
	path    string
	modTime time.Time
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

type createRollback struct {
	dir              string
	metaPath         string
	metaTmpPath      string
	previousMetadata string
	metadataExisted  bool
}

// removeStaleCreateMetadataTemp removes a temp entry left by an older Create or
// a crash. It unlinks the directory entry without following it, so a planted
// metadata.json.tmp symlink cannot redirect a later write outside output.
func removeStaleCreateMetadataTemp(output string) error {
	path := filepath.Join(output, "metadata.json.tmp")
	info, err := os.Lstat(path)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return fmt.Errorf("inspect stale metadata temp: %w", err)
	case info.Mode().IsRegular(), info.Mode()&os.ModeSymlink != 0:
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale metadata temp: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("stale metadata temp path %q is not a regular file or symlink", path)
	}
}

// stageRollbackFile preserves an existing output file without a full copy when
// the filesystem supports hard links. The staged copy lives inside the backup
// directory, so it is on the same filesystem as the destination and can be
// renamed back atomically.
func stageRollbackFile(src, dst string) (bool, error) {
	info, err := os.Lstat(src)
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err != nil:
		return false, err
	case !info.Mode().IsRegular():
		return false, fmt.Errorf("%q is not a regular file", src)
	}

	if err := os.Link(src, dst); err == nil {
		if err := syncFileAtPath(dst); err != nil {
			_ = os.Remove(dst)
			return false, fmt.Errorf("sync linked rollback file: %w", err)
		}
		return true, nil
	}
	if err := copyFileAtomic(src, dst); err != nil {
		return false, err
	}
	return true, nil
}

// stageCreateRollback records the prior visible backup set before any new DB
// snapshot replaces it. A post-promotion metadata failure can then restore the
// exact files that Verify accepted before this Create call.
func stageCreateRollback(output string, snaps []snapshot) (createRollback, error) {
	if err := removeStaleCreateMetadataTemp(output); err != nil {
		return createRollback{}, err
	}
	dir, err := os.MkdirTemp(output, createRollbackPattern)
	if err != nil {
		return createRollback{}, fmt.Errorf("create rollback dir: %w", err)
	}
	rb := createRollback{
		dir:         dir,
		metaPath:    filepath.Join(output, "metadata.json"),
		metaTmpPath: filepath.Join(dir, "metadata.json.tmp"),
	}
	fail := func(err error) (createRollback, error) {
		_ = os.RemoveAll(dir)
		return createRollback{}, err
	}

	for i := range snaps {
		snap := &snaps[i]
		previousPath := filepath.Join(dir, snap.name+".db")
		existed, err := stageRollbackFile(snap.destPath, previousPath)
		if err != nil {
			return fail(fmt.Errorf("stage previous snapshot %s: %w", snap.name, err))
		}
		if existed {
			snap.hadDest = true
			snap.previousPath = previousPath
		}
	}

	info, err := os.Lstat(rb.metaPath)
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return fail(fmt.Errorf("inspect previous metadata: %w", err))
	default:
		if !info.Mode().IsRegular() {
			return fail(fmt.Errorf("previous metadata path %q is not a regular file", rb.metaPath))
		}
		rb.metadataExisted = true
		previousPath := filepath.Join(dir, "metadata.json")
		if _, err := stageRollbackFile(rb.metaPath, previousPath); err != nil {
			return fail(fmt.Errorf("stage previous metadata: %w", err))
		}
		rb.previousMetadata = previousPath
	}
	if err := syncDirFn(dir); err != nil {
		return fail(fmt.Errorf("sync create rollback directory: %w", err))
	}
	if err := syncDirFn(output); err != nil {
		return fail(fmt.Errorf("sync create rollback directory entry: %w", err))
	}
	return rb, nil
}

func rollbackCreate(snaps []snapshot, promoted []int, rb createRollback) error {
	var rollbackErrors []error
	for i := len(promoted) - 1; i >= 0; i-- {
		snap := snaps[promoted[i]]
		var err error
		if snap.hadDest {
			err = os.Rename(snap.previousPath, snap.destPath)
		} else {
			err = os.Remove(snap.destPath)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restore previous %s: %w", snap.name, err))
		}
	}

	var metadataErr error
	switch {
	case rb.previousMetadata != "":
		metadataErr = os.Rename(rb.previousMetadata, rb.metaPath)
	case !rb.metadataExisted:
		metadataErr = os.Remove(rb.metaPath)
		if os.IsNotExist(metadataErr) {
			metadataErr = nil
		}
	}
	if metadataErr != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("restore previous metadata: %w", metadataErr))
	}
	if err := syncDirFn(filepath.Dir(rb.metaPath)); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("sync restored backup set: %w", err))
	}

	_ = os.Remove(rb.metaTmpPath)
	for _, snap := range snaps {
		_ = os.Remove(snap.tmpPath)
	}
	return errors.Join(rollbackErrors...)
}

// recoverInterruptedCreate repairs the narrow crash window in Create where a
// new database snapshot has replaced the visible one but metadata.json still
// describes the previous set. Normal errors run Create's deferred rollback;
// only SIGKILL/power loss can leave this state behind. The hidden rollback
// directory already holds the previous verified files, so Verify (and the next
// Create) can finish the rollback without requiring a manual .pre_restore
// operation.
func (s *BackupService) recoverInterruptedCreate(ctx context.Context, output string) error {
	candidates, err := findCreateRollbackDirs(output)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}

	// A complete set means the crash happened after the metadata rename. The
	// new snapshot is committed and only the rollback directory is stale.
	if _, err := verifyArchiveContents(output); err == nil {
		return removeCreateRollbackDirs(candidates)
	}

	latest := candidates[0]
	hadCommittedSet, err := s.restoreCreateRollback(output, latest.path)
	if err != nil {
		return fmt.Errorf("restore previous set from %q: %w", latest.path, err)
	}
	if !hadCommittedSet {
		return removeCreateRollbackDirs(candidates)
	}
	if _, err := verifyArchiveContents(output); err != nil {
		return fmt.Errorf("verify recovered set from %q: %w", latest.path, err)
	}
	return removeCreateRollbackDirs(candidates)
}

func findCreateRollbackDirs(output string) ([]createRollbackCandidate, error) {
	entries, err := os.ReadDir(output)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list backup output: %w", err)
	}

	candidates := make([]createRollbackCandidate, 0, 1)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), createRollbackPrefix) {
			continue
		}
		path := filepath.Join(output, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect create rollback %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("create rollback path %q is not a directory", path)
		}
		candidates = append(candidates, createRollbackCandidate{path: path, modTime: info.ModTime()})
	}
	slices.SortFunc(candidates, func(a, b createRollbackCandidate) int {
		if order := b.modTime.Compare(a.modTime); order != 0 {
			return order
		}
		return strings.Compare(a.path, b.path)
	})
	return candidates, nil
}

func (s *BackupService) restoreCreateRollback(output, rollbackDir string) (bool, error) {
	var names []string
	hadCommittedSet := true
	rollbackMetaPath := filepath.Join(rollbackDir, "metadata.json")
	metaInfo, err := os.Lstat(rollbackMetaPath)
	switch {
	case err == nil:
		if !metaInfo.Mode().IsRegular() {
			return false, fmt.Errorf("rollback metadata path %q is not a regular file", rollbackMetaPath)
		}
		metadata, err := verifyArchiveContents(rollbackDir)
		if err != nil {
			return false, fmt.Errorf("verify rollback set: %w", err)
		}
		names = metadata.Databases
	case os.IsNotExist(err):
		// A crash during the first Create has no previous complete set. There
		// is no rollback metadata to name the promoted files, so clean up the
		// names from the process that initiated recovery.
		hadCommittedSet = false
		names = make([]string, 0, len(s.dbPaths))
		for name := range s.dbPaths {
			if err := validateDatabaseName(name); err != nil {
				return false, fmt.Errorf("invalid configured database name: %w", err)
			}
			names = append(names, name)
		}
	case err != nil:
		return false, fmt.Errorf("inspect rollback metadata: %w", err)
	}
	slices.Sort(names)

	for _, name := range names {
		src := filepath.Join(rollbackDir, name+".db")
		dst := filepath.Join(output, name+".db")
		restored, err := restoreRollbackFile(src, dst)
		if err != nil {
			return false, fmt.Errorf("restore %s: %w", name, err)
		}
		if !restored {
			if err := removeRollbackDestination(dst); err != nil {
				return false, fmt.Errorf("remove new %s snapshot: %w", name, err)
			}
		}
	}

	metaPath := filepath.Join(output, "metadata.json")
	restored, err := restoreRollbackFile(rollbackMetaPath, metaPath)
	if err != nil {
		return false, fmt.Errorf("restore metadata: %w", err)
	}
	if !restored {
		if err := removeRollbackDestination(metaPath); err != nil {
			return false, fmt.Errorf("remove new metadata: %w", err)
		}
	}
	return hadCommittedSet, nil
}

func restoreRollbackFile(src, dst string) (bool, error) {
	info, err := os.Lstat(src)
	switch {
	case os.IsNotExist(err):
		return false, nil
	case err != nil:
		return false, err
	case !info.Mode().IsRegular():
		return false, fmt.Errorf("rollback artifact %q is not a regular file", src)
	}

	tmpPath := dst + ".restore.tmp"
	if err := copyFileAtomic(src, tmpPath); err != nil {
		return false, err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return false, err
	}
	if err := syncDirFn(filepath.Dir(dst)); err != nil {
		return true, fmt.Errorf("sync restored file: %w", err)
	}
	return true, nil
}

func removeRollbackDestination(path string) error {
	info, err := os.Lstat(path)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	case info.IsDir():
		return fmt.Errorf("%q is a directory", path)
	default:
		return os.Remove(path)
	}
}

func removeCreateRollbackDirs(candidates []createRollbackCandidate) error {
	var removeErrors []error
	outputDirs := make(map[string]struct{}, 1)
	for _, candidate := range candidates {
		info, err := os.Lstat(candidate.path)
		switch {
		case os.IsNotExist(err):
			continue
		case err != nil:
			removeErrors = append(removeErrors, fmt.Errorf("inspect %q: %w", candidate.path, err))
			continue
		case info.Mode()&os.ModeSymlink != 0 || !info.IsDir():
			removeErrors = append(removeErrors, fmt.Errorf("rollback path %q is not a directory", candidate.path))
			continue
		}
		if err := os.RemoveAll(candidate.path); err != nil {
			removeErrors = append(removeErrors, fmt.Errorf("remove %q: %w", candidate.path, err))
			continue
		}
		outputDirs[filepath.Dir(candidate.path)] = struct{}{}
	}
	for dir := range outputDirs {
		if err := syncDirFn(dir); err != nil {
			removeErrors = append(removeErrors, fmt.Errorf("sync %q: %w", dir, err))
		}
	}
	return errors.Join(removeErrors...)
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
