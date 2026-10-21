package lottery

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

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
	output, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return nil, 0, fmt.Errorf("marshal records: %w", err)
	}
	return output, len(records), nil
}

// ImportResult reports what ImportRecords did with a document.
type ImportResult struct {
	// Total is the number of records the document carried.
	Total int
	// Imported is the number stored.
	Imported int
	// Rejected lists, by 0-based index, every record that was NOT stored:
	// it failed validation, collided with an existing draw, or could not
	// be written.
	Rejected []int
}

// Partial reports whether some records were stored and some rejected.
func (r ImportResult) Partial() bool { return r.Imported > 0 && len(r.Rejected) > 0 }

// ImportRecords stores the draws of a portable document. It is the single
// implementation behind `aurora lottery import` and the desktop shell's native
// Import action.
//
// The document is untrusted input, so every record is validated first. A
// record whose ID already exists is skipped instead of replaced: IDs are
// deterministic (sha256 of seed + VRF output) and the repository's Save is an
// INSERT OR REPLACE, so a re-import would otherwise silently overwrite the
// stored draw's Verified flag and audit trail (TASK-069, ISS-061).
//
// A malformed document is a hard error (nothing is written). Per-record
// rejections are reported in ImportResult.Rejected, not as an error: callers
// decide how loudly to report a partial import (the CLI exits non-zero, the
// desktop shows a warning).
func ImportRecords(repo lottery.Repository, document []byte) (ImportResult, error) {
	var records []lottery.LotteryRecord
	if err := json.Unmarshal(document, &records); err != nil {
		return ImportResult{}, fmt.Errorf("parse document: %w", err)
	}

	result := ImportResult{Total: len(records)}
	for i, record := range records {
		if err := record.Validate(); err != nil {
			result.Rejected = append(result.Rejected, i)
			continue
		}
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
