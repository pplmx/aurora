package lotteryapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/utils"
)

// MaxImportFileSize bounds an import document. An export of the whole ledger is
// megabytes; a much larger file is a mistake (or not an export at all), and
// reading it whole would grow the process by the file's size before the parser
// rejects it. The CLI and the desktop both read through ReadImportFile so the
// bound cannot drift between them.
const MaxImportFileSize int64 = 64 << 20

// ErrImportFileTooLarge marks a document above MaxImportFileSize. It is a
// sentinel so a presentation layer can name the limit instead of echoing the
// raw size.
var ErrImportFileTooLarge = errors.New("import file too large")

// ReadImportFile reads a lottery export, refusing anything above
// MaxImportFileSize. The read is bounded rather than stat-then-read: a file
// that grows while it is read must not slip past the limit.
func ReadImportFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, MaxImportFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxImportFileSize {
		return nil, fmt.Errorf("%w: max %d bytes", ErrImportFileTooLarge, MaxImportFileSize)
	}
	return data, nil
}

// ErrInvalidImportDocument marks a file that is not a valid portable lottery
// export. Keeping it separate from per-record rejections lets callers explain
// that nothing was read before any record could be considered.
var ErrInvalidImportDocument = errors.New("invalid lottery import document")

// ImportDocumentError keeps the parser wording from json.Unmarshal intact
// while still exposing ErrInvalidImportDocument to errors.Is. The explicit
// detail also lets presentation layers localize the stable context without
// parsing a formatted error string.
type ImportDocumentError struct {
	detail string
}

func (e *ImportDocumentError) Error() string {
	return e.LocalizationDetail()
}

func (e *ImportDocumentError) Unwrap() error {
	return ErrInvalidImportDocument
}

// LocalizationDetail returns the stable, non-localized detail for a
// presentation layer to translate alongside the sentinel.
func (e *ImportDocumentError) LocalizationDetail() string {
	return "parse document: " + e.detail
}

// ExportRecords serializes every stored draw as the portable JSON document
// `aurora lottery export` writes and `aurora lottery import` reads, and
// returns how many draws the document carries (for the caller's message).
//
// The document is the repository's records verbatim (indented), not the
// blockchain blocks: history/stats/verify all read lottery_records, and an
// imported draw only ever reaches that table.
func ExportRecords(repo lottery.Repository) (document []byte, count int, err error) {
	records, err := repo.GetAll()
	if err != nil {
		return nil, 0, fmt.Errorf("read records: %w", err)
	}
	// A repository with no draws returns a nil slice, which json.MarshalIndent
	// would encode as `null`. The portable format is always a JSON array, so
	// an empty export must be `[]` - a null document is not something the
	// importer can tell apart from an empty list.
	if records == nil {
		records = []*lottery.LotteryRecord{}
	}
	output, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return nil, 0, fmt.Errorf("marshal records: %w", err)
	}
	return output, len(records), nil
}

// WriteRecordsToFile writes the portable document of repo to path. It is the
// one export entry point for the CLI and the desktop menu, so both apply the
// same safety checks: the destination may not be the live database (or one of
// its sidecars), and the write is atomic.
func WriteRecordsToFile(repo lottery.Repository, path string) (count int, err error) {
	if provider, ok := repo.(interface{ DBPath() string }); ok {
		if err := refuseLiveDatabaseDestination(provider.DBPath(), path); err != nil {
			return 0, err
		}
	}
	document, count, err := ExportRecords(repo)
	if err != nil {
		return 0, err
	}
	// The document carries participant names and their draws, so it gets the
	// same owner-only mode as the database, the logs and the backups; sharing it
	// is an explicit copy, not the default state of the file.
	if err := utils.WriteFileAtomic(path, document, 0o600); err != nil {
		return 0, err
	}
	return count, nil
}

// refuseLiveDatabaseDestination rejects a destination that would replace the
// database being exported. os.SameFile catches a symlink or hard-link alias;
// the cleaned path comparison also covers a sidecar that does not exist yet
// but whose exact name would land on a live SQLite sidecar.
func refuseLiveDatabaseDestination(dbPath, dest string) error {
	if dbPath == "" {
		return nil
	}
	path, ephemeral, err := utils.SQLiteFilePath(dbPath)
	if err != nil {
		return fmt.Errorf("resolve live database path: %w", err)
	}
	if ephemeral {
		return nil
	}
	livePaths := append([]string{path}, utils.SQLiteSidecarPaths(path)...)
	// SQLite names sidecars beside the symlink target, not the symlink path.
	// Include the resolved database and its sidecars so exporting to the real
	// WAL, SHM, or rollback journal cannot bypass the guard.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		livePaths = append(livePaths, resolved)
		livePaths = append(livePaths, utils.SQLiteSidecarPaths(resolved)...)
	}
	for _, live := range livePaths {
		if sameExistingFile(live, dest) || sameCleanPath(live, dest) {
			return fmt.Errorf("refusing to export over the live database file %q", live)
		}
	}
	return nil
}

func sameExistingFile(a, b string) bool {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(aInfo, bInfo)
}

func sameCleanPath(a, b string) bool {
	aAbs, aErr := filepath.Abs(a)
	bAbs, bErr := filepath.Abs(b)
	if aErr != nil || bErr != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(aAbs) == filepath.Clean(bAbs)
}

// ImportResult reports what ImportRecords did with a document.
type ImportResult struct {
	// Total is the number of records the document carried.
	Total int `json:"total"`
	// Imported is the number stored.
	Imported int `json:"imported"`
	// Rejected lists, by 0-based index, every record that was NOT stored:
	// it failed validation, collided with an existing draw, or could not
	// be written.
	Rejected []int `json:"rejected"`
}

// importDetail turns a decode failure into the sentence a reader needs. A
// top-level shape mismatch would otherwise leak the Go type name ("cannot
// unmarshal object into Go value of type []lottery.LotteryRecord") into the CLI
// and the window. Errors that name a field keep json's wording, because the
// field is what the reader has to fix.
func importDetail(err error) string {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field == "" {
		return fmt.Sprintf("expected a JSON array of lottery records, got %s", typeErr.Value)
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "not a valid JSON document: " + syntaxErr.Error()
	}
	return err.Error()
}

// ImportRecords stores the draws of a portable document. It is the single
// implementation behind `aurora lottery import` and the desktop shell's native
// Import action.
//
// The document is untrusted input, so every record is validated and its stored
// winners, output, proof and public key are checked with VerifyRecord before
// it can be stored. The serialized Verified bit is not trusted: it is derived
// from the verifier, and records without proof-verifiable evidence remain
// unverified. A record whose ID already exists is skipped instead of replaced:
// IDs are deterministic (sha256 of seed + VRF output) and the repository's Save
// is an INSERT OR REPLACE, so a re-import would otherwise silently overwrite
// the stored draw's Verified flag and audit trail (TASK-069, ISS-061).
//
// A malformed document is a hard error (nothing is written). Per-record
// rejections are reported in ImportResult.Rejected, not as an error: callers
// decide how loudly to report a partial import (the CLI exits non-zero, the
// desktop shows a warning).
func ImportRecords(repo lottery.Repository, document []byte) (ImportResult, error) {
	var records []lottery.LotteryRecord
	if err := json.Unmarshal(document, &records); err != nil {
		return ImportResult{}, &ImportDocumentError{detail: importDetail(err)}
	}
	// json.Unmarshal accepts a literal null into a slice and leaves it nil.
	// Treating that as an empty document reported a successful import of 0
	// records for a file that is not the portable array format at all.
	if records == nil {
		return ImportResult{}, &ImportDocumentError{
			detail: "expected a JSON array of lottery records, got null",
		}
	}
	result := ImportResult{Total: len(records)}
	for i, record := range records {
		if err := record.Validate(); err != nil {
			result.Rejected = append(result.Rejected, i)
			continue
		}
		report := VerifyRecord(&record)
		if !report.IntegrityOK {
			result.Rejected = append(result.Rejected, i)
			continue
		}
		record.Verified = report.ProofVerified
		if existing, err := repo.GetByID(record.ID); err == nil && existing != nil {
			result.Rejected = append(result.Rejected, i)
			continue
		} else if err != nil && !errors.Is(err, lottery.ErrNotFound) {
			result.Rejected = append(result.Rejected, i)
			continue
		}
		if err := repo.Save(&record); err != nil {
			result.Rejected = append(result.Rejected, i)
			continue
		}
		result.Imported++
	}
	return result, nil
}
