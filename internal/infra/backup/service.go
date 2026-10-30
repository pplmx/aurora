package backup

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/utils"
)

type BackupService struct {
	dbPaths   map[string]string
	configErr error
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
	resolvedPaths, configErr := resolveBackupPaths(dbPaths)
	return &BackupService{
		dbPaths:         resolvedPaths,
		configErr:       configErr,
		renameFn:        os.Rename,
		writeFileFn:     os.WriteFile,
		removeSidecarFn: os.Remove,
	}
}

// resolveBackupPaths converts every configured SQLite location to the local
// file it names before any backup or restore filesystem operation. Keeping the
// raw file: URI here would make os.Stat look for a literal "file:" directory
// and would let a memory database appear as a missing source file.
func resolveBackupPaths(dbPaths map[string]string) (map[string]string, error) {
	resolved := make(map[string]string, len(dbPaths))
	var configErrors []error
	for name, location := range dbPaths {
		path, ephemeral, err := utils.SQLiteFilePath(location)
		switch {
		case err != nil:
			// Preserve the database name for archive-set validation even
			// when its configured location is malformed.
			resolved[name] = location
			configErrors = append(configErrors, fmt.Errorf("database %q: %w", name, err))
		case ephemeral:
			resolved[name] = location
			configErrors = append(configErrors, fmt.Errorf(
				"database %q is not disk-backed: %q",
				name,
				location,
			))
		default:
			resolved[name] = path
		}
	}
	return resolved, errors.Join(configErrors...)
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

func validateUniqueArchiveDatabaseNames(archiveNames []string) error {
	seen := make(map[string]struct{}, len(archiveNames))
	for _, name := range archiveNames {
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("backup metadata lists database %q more than once", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

// validateArchiveDatabaseSet keeps Verify and Restore on one definition of a
// usable archive. A checksum-consistent metadata file can still be unusable:
// duplicate names are ambiguous, and a configured installation can only
// restore the exact set of databases it owns.
func (s *BackupService) validateArchiveDatabaseSet(archiveNames []string) error {
	if err := validateUniqueArchiveDatabaseNames(archiveNames); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(archiveNames))
	for _, name := range archiveNames {
		seen[name] = struct{}{}
	}
	if len(s.dbPaths) == 0 {
		return nil
	}
	if len(archiveNames) != len(s.dbPaths) {
		return fmt.Errorf(
			"backup contains %d databases but %d are configured",
			len(archiveNames), len(s.dbPaths),
		)
	}
	for name := range seen {
		if _, configured := s.dbPaths[name]; !configured {
			return fmt.Errorf("backup database %q is not configured", name)
		}
	}
	for name := range s.dbPaths {
		if _, present := seen[name]; !present {
			return fmt.Errorf("configured database %q is missing from backup metadata", name)
		}
	}
	return nil
}
