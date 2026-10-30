package lotteryapp

import (
	"errors"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

// scriptedRepo is a lottery.Repository whose every method answers with a
// canned result. The resolution order in resolveRecord is the contract the
// CLI, desktop and REST all inherit, and each step has its own failure and
// empty-result spelling, so the tests need to steer them independently.
type scriptedRepo struct {
	byID      *lottery.LotteryRecord
	byIDErr   error
	byHeight  []*lottery.LotteryRecord
	heightErr error
	all       []*lottery.LotteryRecord
	allErr    error

	getByIDCalls    int
	getByHeightCall int
	getAllCalls     int
}

func (r *scriptedRepo) Save(*lottery.LotteryRecord) error { return nil }

func (r *scriptedRepo) GetByID(string) (*lottery.LotteryRecord, error) {
	r.getByIDCalls++
	return r.byID, r.byIDErr
}

func (r *scriptedRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	r.getAllCalls++
	return r.all, r.allErr
}

func (r *scriptedRepo) Count() (int, error) {
	return len(r.all), r.allErr
}

func (r *scriptedRepo) List(limit, offset int) ([]*lottery.LotteryRecord, error) {
	if offset >= len(r.all) {
		return nil, r.allErr
	}
	records := r.all[offset:]
	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}
	return records, r.allErr
}
func (r *scriptedRepo) ListMatching(term string, limit, offset int) ([]*lottery.LotteryRecord, error) {
	return r.List(limit, offset)
}

func (r *scriptedRepo) GetByBlockHeight(int64) ([]*lottery.LotteryRecord, error) {
	r.getByHeightCall++
	return r.byHeight, r.heightErr
}

// TestVerifyInput_ResolvesByExactID pins the first resolution step: a stored
// record wins before any height or substring interpretation, which is what
// keeps an all-digit ID from being read as a block height (TASK-200, ISS-196).
func TestVerifyInput_ResolvesByExactID(t *testing.T) {
	record := verifiableLotteryRecord(t)
	repo := &scriptedRepo{byID: record}

	report, err := VerifyInput(repo, record.ID)
	if err != nil {
		t.Fatalf("VerifyInput: %v", err)
	}
	if report.Record != record {
		t.Fatal("the exact-ID step must return the stored record itself")
	}
	if !report.IntegrityOK {
		t.Fatalf("canonical draw must verify: %+v", report.Checks)
	}
	if repo.getByHeightCall != 0 || repo.getAllCalls != 0 {
		t.Fatal("a hit on the first step must not query later steps")
	}
}

// TestVerifyInput_NilRecordFallsThrough covers the second legal "not found"
// spelling: a repository that answers (nil, nil) instead of the ErrNotFound
// sentinel. Treating it as a hit handed VerifyRecord a nil pointer and the
// verification panicked, so every caller (CLI verify, desktop verifier, REST)
// crashed on a missing row from such a repository.
func TestVerifyInput_NilRecordFallsThrough(t *testing.T) {
	record := verifiableLotteryRecord(t)
	repo := &scriptedRepo{byID: nil, all: []*lottery.LotteryRecord{record}}

	report, err := VerifyInput(repo, record.Seed)
	if err != nil {
		t.Fatalf("a nil record must fall through to the substring step: %v", err)
	}
	if report.Record != record {
		t.Fatalf("expected the substring step to resolve %s", record.ID)
	}
}

func TestVerifyInput_NilRecordWithoutMatchIsNotFound(t *testing.T) {
	repo := &scriptedRepo{byID: nil, all: nil}

	_, err := VerifyInput(repo, "no-such-draw")
	if err == nil {
		t.Fatal("an unresolvable reference must report not found")
	}
	if !strings.Contains(err.Error(), "lottery not found") {
		t.Fatalf("error must say the record is missing, got %v", err)
	}
}

// TestVerifyInput_ResolvesByBlockHeight pins the second step and its
// empty-result spelling: a known height with no draws is "not found", not a
// nil report.
func TestVerifyInput_ResolvesByBlockHeight(t *testing.T) {
	record := verifiableLotteryRecord(t)
	record.BlockHeight = 42
	repo := &scriptedRepo{byIDErr: lottery.ErrNotFound, byHeight: []*lottery.LotteryRecord{record}}

	report, err := VerifyInput(repo, "42")
	if err != nil {
		t.Fatalf("VerifyInput by height: %v", err)
	}
	if report.Record != record {
		t.Fatal("the height step must return the draw stored at that height")
	}
}

func TestVerifyInput_EmptyHeightResultIsNotFound(t *testing.T) {
	repo := &scriptedRepo{byIDErr: lottery.ErrNotFound, byHeight: []*lottery.LotteryRecord{}}

	_, err := VerifyInput(repo, "42")
	if err == nil || !strings.Contains(err.Error(), "lottery not found") {
		t.Fatalf("a height with no draws must report not found, got %v", err)
	}
	if repo.getAllCalls != 0 {
		t.Fatal("a numeric reference must not fall through to the substring step")
	}
}

// TestVerifyInput_HeightListSkipsNilEntries: the height lookup returns a
// slice, so one corrupt entry must not shadow a usable one behind it.
func TestVerifyInput_HeightListSkipsNilEntries(t *testing.T) {
	record := verifiableLotteryRecord(t)
	repo := &scriptedRepo{
		byIDErr:  lottery.ErrNotFound,
		byHeight: []*lottery.LotteryRecord{nil, record},
	}

	report, err := VerifyInput(repo, "7")
	if err != nil {
		t.Fatalf("VerifyInput: %v", err)
	}
	if report.Record != record {
		t.Fatal("a nil entry must be skipped, not returned")
	}
}

func TestVerifyInput_HeightListOfOnlyNilIsNotFound(t *testing.T) {
	repo := &scriptedRepo{byIDErr: lottery.ErrNotFound, byHeight: []*lottery.LotteryRecord{nil}}

	if _, err := VerifyInput(repo, "7"); err == nil {
		t.Fatal("a height list of only nil entries must report not found")
	}
}

// TestVerifyInput_ResolvesBySubstring pins the third step for both fields the
// help text promises: an ID fragment and a seed fragment.
//
// The record is built from literals rather than from DrawWinners: the real ID
// is a hash whose characters depend on a freshly generated key pair, so an
// 8-character fragment was occasionally all digits and then parsed as a block
// height instead of reaching the substring step (measured: the test failed
// with "lottery not found: 31506309").
func TestVerifyInput_ResolvesBySubstring(t *testing.T) {
	record := &lottery.LotteryRecord{
		ID:   "ab12cd34ef567890",
		Seed: "address-integrity",
	}
	repo := &scriptedRepo{byIDErr: lottery.ErrNotFound, all: []*lottery.LotteryRecord{record}}

	for _, ref := range []string{"ab12cd34", "address-integrity"} {
		report, err := VerifyInput(repo, ref)
		if err != nil {
			t.Fatalf("VerifyInput(%q): %v", ref, err)
		}
		if report.Record != record {
			t.Fatalf("substring %q must resolve the draw", ref)
		}
	}
}

// TestVerifyInput_SubstringSkipsNilEntries keeps a corrupt history slice from
// panicking the third step the same way the first one did.
func TestVerifyInput_SubstringSkipsNilEntries(t *testing.T) {
	// A literal id keeps the reference a hash prefix: an 8-character slice of
	// a real ID was sometimes all digits and parsed as a block height instead
	// of reaching the substring step (measured, same flaw as the test above).
	record := &lottery.LotteryRecord{ID: "ab12cd34ef567890", Seed: "address-integrity"}
	repo := &scriptedRepo{
		byIDErr: lottery.ErrNotFound,
		all:     []*lottery.LotteryRecord{nil, record},
	}

	report, err := VerifyInput(repo, "ab12cd34")
	if err != nil {
		t.Fatalf("VerifyInput: %v", err)
	}
	if report.Record != record {
		t.Fatal("a nil history entry must be skipped")
	}
}

// TestVerifyInput_PropagatesReadFailures pins that a genuine storage failure
// is never folded into "not found": scripts distinguish a missing draw from a
// broken database by exit code and message.
func TestVerifyInput_PropagatesReadFailures(t *testing.T) {
	boom := errors.New("disk on fire")
	tests := []struct {
		name   string
		repo   *scriptedRepo
		ref    string
		detail string
	}{
		{
			name:   "exact id read",
			repo:   &scriptedRepo{byIDErr: boom},
			ref:    "abc",
			detail: "failed to read lottery record",
		},
		{
			name:   "height read",
			repo:   &scriptedRepo{byIDErr: lottery.ErrNotFound, heightErr: boom},
			ref:    "42",
			detail: "failed to read by block height",
		},
		{
			name:   "history read",
			repo:   &scriptedRepo{byIDErr: lottery.ErrNotFound, allErr: boom},
			ref:    "abc",
			detail: "failed to read history",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyInput(tt.repo, tt.ref)
			if err == nil {
				t.Fatal("a read failure must be returned")
			}
			if !errors.Is(err, boom) {
				t.Fatalf("the storage error must stay in the chain, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.detail) {
				t.Fatalf("error must name the failing step %q, got %v", tt.detail, err)
			}
		})
	}
}

// TestVerifyInput_TamperedRecordIsReportedNotResolvedTwice keeps the two
// halves of the entry point distinct: resolution finds the row, verification
// judges it, and a failed judgement is a report (IntegrityOK=false) rather
// than an error.
func TestVerifyInput_TamperedRecordIsReported(t *testing.T) {
	record := verifiableLotteryRecord(t)
	record.Winners = record.Winners[:1]
	repo := &scriptedRepo{byID: record}

	report, err := VerifyInput(repo, record.ID)
	if err != nil {
		t.Fatalf("a tampered record must still resolve: %v", err)
	}
	if report.IntegrityOK {
		t.Fatal("a tampered record must report a failed integrity check")
	}
	if !hasVerifyFailure(report, "winner/address") {
		t.Fatalf("report must name the mismatch: %+v", report.Checks)
	}
}

// TestVerifyInput_SubstringPicksTheNewestMatch pins the ambiguity rule: a
// reference that matches several draws resolves to the first one in the
// repository's newest-first history, so the same input never flips between two
// reports on different machines.
func TestVerifyInput_SubstringPicksTheNewestMatch(t *testing.T) {
	newest := &lottery.LotteryRecord{ID: "bb11cc22dd334455", Seed: "weekly-2026-10"}
	older := &lottery.LotteryRecord{ID: "aa11cc22dd334466", Seed: "weekly-2026-09"}
	repo := &scriptedRepo{
		byIDErr: lottery.ErrNotFound,
		all:     []*lottery.LotteryRecord{newest, older},
	}

	report, err := VerifyInput(repo, "weekly-2026")
	if err != nil {
		t.Fatalf("VerifyInput: %v", err)
	}
	if report.Record != newest {
		t.Fatalf("an ambiguous reference must resolve the newest match, got %q", report.Record.Seed)
	}
}
