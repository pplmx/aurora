package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type BackupService struct {
	dbPaths map[string]string
	// renameFn is injectable so tests can exercise a failure after one DB
	// snapshot has been promoted but before the complete set is in place.
	renameFn func(string, string) error
	// writeFileFn is injectable so tests can fail the metadata write after all
	// database snapshots have been promoted.
	writeFileFn func(string, []byte, os.FileMode) error
	// removeSidecarFn is injectable so tests can exercise post-promotion rollback.
	removeSidecarFn func(string) error
}

// syncFileFn is injectable so tests can prove that copied database bytes are
// durable before Restore or its rollback renames the temp over a live file.
var syncFileFn = func(f *os.File) error {
	return f.Sync()
}

// syncDirFn makes a rename or unlink durable. Windows cannot fsync a directory
// handle through os.File, so its rename durability remains delegated to the
// platform; every Unix release flushes the containing directory.
var syncDirFn = func(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return err
	}
	return dir.Close()
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
	return &BackupService{
		dbPaths:         dbPaths,
		renameFn:        os.Rename,
		writeFileFn:     os.WriteFile,
		removeSidecarFn: os.Remove,
	}
}

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
	if err := s.verify(ctx, output); err == nil {
		return removeCreateRollbackDirs(candidates)
	}

	latest := candidates[0]
	if err := s.restoreCreateRollback(output, latest.path); err != nil {
		return fmt.Errorf("restore previous set from %q: %w", latest.path, err)
	}
	if err := s.verify(ctx, output); err != nil {
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

func (s *BackupService) restoreCreateRollback(output, rollbackDir string) error {
	names := make([]string, 0, len(s.dbPaths))
	for name := range s.dbPaths {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		src := filepath.Join(rollbackDir, name+".db")
		dst := filepath.Join(output, name+".db")
		restored, err := restoreRollbackFile(src, dst)
		if err != nil {
			return fmt.Errorf("restore %s: %w", name, err)
		}
		if !restored {
			if err := removeRollbackDestination(dst); err != nil {
				return fmt.Errorf("remove new %s snapshot: %w", name, err)
			}
		}
	}

	metaPath := filepath.Join(output, "metadata.json")
	restored, err := restoreRollbackFile(filepath.Join(rollbackDir, "metadata.json"), metaPath)
	if err != nil {
		return fmt.Errorf("restore metadata: %w", err)
	}
	if !restored {
		if err := removeRollbackDestination(metaPath); err != nil {
			return fmt.Errorf("remove new metadata: %w", err)
		}
	}
	return nil
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
		File:          output,
		Size:          totalSize,
		Checksum:      checksum,
		SchemaVersion: schemaVersion,
	}, nil
}

// restrictBackupFile makes a database snapshot or its metadata owner-only.
// SQLite's VACUUM INTO creates the snapshot with the process umask, so the
// file itself was 0644 under the common 022 default even when the caller
// selected a private directory. Windows backup directories inherit the user's
// ACL and do not use these Unix mode bits.
func restrictBackupFile(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, 0o600)
}

// restrictBackupDir tightens the hidden restore staging directory, including
// directories left behind by older releases at 0755. A deliberate symlink is
// rejected so Chmod cannot retarget an unrelated directory.
func restrictBackupDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is a symlink", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, 0o700)
}

// mkdirAllDurable creates path and flushes each new directory entry in its
// parent. Syncing only the leaf directory persists its contents, not the leaf
// itself: a desktop backup creates a new timestamp directory, so a power loss
// could otherwise remove the entire successfully reported backup.
func mkdirAllDurable(path string, perm os.FileMode) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve directory path: %w", err)
	}
	absolute = filepath.Clean(absolute)

	var missing []string
	for current := absolute; ; {
		info, statErr := os.Stat(current)
		if statErr == nil {
			if !info.IsDir() {
				return fmt.Errorf("%q is not a directory", current)
			}
			break
		}
		if !os.IsNotExist(statErr) {
			return fmt.Errorf("inspect directory %q: %w", current, statErr)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	if err := os.MkdirAll(absolute, perm); err != nil {
		return err
	}
	for i := len(missing) - 1; i >= 0; i-- {
		parent := filepath.Dir(missing[i])
		if err := syncDirFn(parent); err != nil {
			return fmt.Errorf("sync parent directory for %q: %w", missing[i], err)
		}
	}
	return nil
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

// vacuumInto writes an online (VACUUM INTO) snapshot of the SQLite DB at src
// into dest. The result is a complete, self-contained database file including
// committed WAL frames — unlike a bare .db file copy, which can miss frames
// committed but not yet checkpointed. dest must not exist; callers remove a
// stale file first.
func vacuumInto(src, dest string) error {
	// Hardened DSN so VACUUM INTO against a DB an API/CLI process holds the
	// write lock on waits instead of failing instantly (migrate fix class,
	// TASK-170/ISS-165).
	db, err := sql.Open("sqlite", sqlite.DSN(src))
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("VACUUM INTO " + sqlLiteral(dest)); err != nil {
		return err
	}
	return nil
}

// querySchemaVersion reads schema_migrations.version from a single SQLite DB.
// Returns 0 if the table is missing or the file can't be opened; in either
// case we treat that DB as "version 0" and let the caller decide the overall
// result (typically the highest across all DBs wins).
func querySchemaVersion(path string) uint {
	// Refuse to open a source that doesn't exist: SQLite creates a missing
	// database file on first use, so probing a misconfigured/missing source
	// here would silently materialize an empty DB and let Create "successfully"
	// back it up instead of surfacing the bad source (TASK-109, ISS-101).
	if _, err := os.Stat(path); err != nil {
		return 0
	}
	db, err := sql.Open("sqlite", sqlite.DSN(path))
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

// validateDatabaseName rejects a database name read from backup metadata that
// would escape the backup directory when joined onto it. Backup archives are
// operator-supplied (possibly hostile: `backup verify/restore` runs on
// whatever archive is given), so a name like "../../host" must not turn the
// later filepath.Join(backupPath, name+".db") into an arbitrary host path
// that Verify would stat/hash/open (TASK-135, ISS-126).
func validateDatabaseName(name string) error {
	if name == "" {
		return errors.New("empty database name")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid database name %q", name)
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "\x00") {
		return fmt.Errorf("invalid database name %q: must be a plain filename", name)
	}
	return nil
}

func (s *BackupService) Verify(ctx context.Context, backupPath string) error {
	if err := s.recoverInterruptedCreate(ctx, backupPath); err != nil {
		return fmt.Errorf("recover interrupted backup create: %w", err)
	}
	return s.verify(ctx, backupPath)
}

func (s *BackupService) verify(ctx context.Context, backupPath string) error {
	metaPath := filepath.Join(backupPath, "metadata.json")
	metaInfo, err := os.Lstat(metaPath)
	if err != nil {
		return fmt.Errorf("metadata not found: %w", err)
	}
	if !metaInfo.Mode().IsRegular() {
		return fmt.Errorf("metadata path %q is not a regular file", metaPath)
	}
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
		if err := validateDatabaseName(name); err != nil {
			return fmt.Errorf("invalid database name in backup metadata: %w", err)
		}
	}

	for _, name := range metadata.Databases {
		dbPath := filepath.Join(backupPath, name+".db")
		dbInfo, err := os.Lstat(dbPath)
		if err != nil {
			return fmt.Errorf("missing database file: %s", name)
		}
		if !dbInfo.Mode().IsRegular() {
			return fmt.Errorf("database file %q is not a regular file", dbPath)
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
			db, err := sql.Open("sqlite", sqlite.DSN(dbPath))
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
	metaInfo, err := os.Lstat(metaPath)
	if err != nil {
		return fmt.Errorf("read metadata: %w", err)
	}
	if !metaInfo.Mode().IsRegular() {
		return fmt.Errorf("read metadata: %q is not a regular file", metaPath)
	}
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("read metadata: %w", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(metaData, &metadata); err != nil {
		return fmt.Errorf("parse metadata: %w", err)
	}

	// Defense in depth: Verify already rejects hostile names, but Restore
	// re-reads the archive metadata independently — validate again so a
	// future caller that skips Verify cannot reach filepath.Join with a
	// traversal name (TASK-135, ISS-126).
	for _, name := range metadata.Databases {
		if err := validateDatabaseName(name); err != nil {
			return fmt.Errorf("invalid database name in backup metadata: %w", err)
		}
	}

	operations, err := s.planRestore(backupPath, metadata.Databases)
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
}

// planRestore validates every database before any live file is read or
// replaced. The archive and the running configuration must describe the same
// set: silently restoring a subset would leave a multi-store installation with
// mixed generations even though Restore returned success.
func (s *BackupService) planRestore(backupDir string, archiveNames []string) ([]restoreOperation, error) {
	if len(archiveNames) != len(s.dbPaths) {
		return nil, fmt.Errorf(
			"backup contains %d databases but %d are configured",
			len(archiveNames), len(s.dbPaths),
		)
	}

	seen := make(map[string]struct{}, len(archiveNames))
	for _, name := range archiveNames {
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf("backup metadata lists database %q more than once", name)
		}
		if _, configured := s.dbPaths[name]; !configured {
			return nil, fmt.Errorf("backup database %q is not configured", name)
		}
		seen[name] = struct{}{}
	}

	names := make([]string, 0, len(s.dbPaths))
	for name := range s.dbPaths {
		if _, present := seen[name]; !present {
			return nil, fmt.Errorf("configured database %q is missing from backup metadata", name)
		}
		names = append(names, name)
	}
	slices.Sort(names)

	operations := make([]restoreOperation, 0, len(names))
	for _, name := range names {
		destPath := s.dbPaths[name]
		backupDBPath := filepath.Join(backupDir, name+".db")
		backupInfo, err := os.Lstat(backupDBPath)
		if err != nil {
			return nil, fmt.Errorf("stat backup %s: %w", name, err)
		}
		if !backupInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("backup database %q is not a regular file", backupDBPath)
		}

		op := restoreOperation{
			name:       name,
			backupPath: backupDBPath,
			destPath:   destPath,
			tmpPath:    destPath + ".tmp",
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
		if err := copyFileAtomic(op.backupPath, op.tmpPath); err != nil {
			return fmt.Errorf("prepare restored %s: %w", op.name, err)
		}
	}
	if err := syncDirFn(preRestoreDir); err != nil {
		return fmt.Errorf("sync pre-restore directory: %w", err)
	}
	return nil
}

func copyFileAtomic(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		_ = src.Close()
		return fmt.Errorf("remove stale temp: %w", err)
	}
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		_ = src.Close()
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = src.Close()
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return fmt.Errorf("copy: %w", err)
	}
	if err := src.Close(); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return fmt.Errorf("close source: %w", err)
	}
	if err := syncFileFn(dst); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := restrictBackupFile(dstPath); err != nil {
		_ = os.Remove(dstPath)
		return fmt.Errorf("restrict temp: %w", err)
	}
	return nil
}

func syncFileAtPath(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := syncFileFn(f); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
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

		// A stale WAL/SHM from the pre-restore DB would be replayed against
		// the new archive on next open (v1.75, ISS-082).
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
	for _, ext := range []string{"-wal", "-shm"} {
		if err := remove(destPath + ext); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale %s for %s: %w", ext, name, err)
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

		for _, ext := range []string{"-wal", "-shm"} {
			if err := os.Remove(op.destPath + ext); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove rolled-back %s%s: %w", op.name, ext, err))
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
