package lotteryapp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

// ErrVerificationFailed means the record was found but its contents do not
// hold up: a malformed VRF proof/output, an empty winners list, winners that
// do not match the VRF output, a missing or altered winner address, a malformed
// public key, or a proof that does not verify against it. Callers turn this
// into a non-zero exit (CLI) or an explicit failure report (desktop/REST/import).
var ErrVerificationFailed = errors.New("lottery record failed integrity check")

// VerifyReport is the structured outcome of VerifyRecord. It carries the
// record plus one human-readable line per check, so every surface (CLI text,
// desktop panel, JSON) can render the same evidence without re-deriving it.
type VerifyReport struct {
	// Record is the draw that was verified.
	Record *lottery.LotteryRecord `json:"record"`
	// Checks are the individual findings, in evaluation order. The last one
	// decides IntegrityOK.
	Checks []VerifyCheck `json:"checks"`
	// IntegrityOK reports whether every check passed.
	IntegrityOK bool `json:"integrity_ok"`
	// ProofVerified reports whether the VRF proof itself was checked against
	// a stored public key (false for pre-v1.31 draws, which have none).
	ProofVerified bool `json:"proof_verified"`
}

// VerifyCheck is one finding.
type VerifyCheck struct {
	// Status is "ok" or "fail".
	Status string `json:"status"`
	// Message explains the finding.
	Message string `json:"message"`
}

// Status values for VerifyCheck.
const (
	VerifyOK   = "ok"
	VerifyFail = "fail"
)

// VerifyInput resolves a user-supplied lottery reference to a stored record and
// verifies it. The reference may be a lottery ID, a whole block height, or a
// substring of an ID or seed.
//
// Resolution order matters and is pinned by tests:
//  1. exact ID match — the common case (users paste an ID from create/history);
//  2. the whole input as a decimal block height;
//  3. a substring match on ID or seed, for convenience.
//
// The exact-ID lookup comes first because a lottery ID is a 16-hex-char hash
// prefix that may itself be all digits: parsing a numeric prefix as a height
// first would report "lottery not found" for an existing draw (TASK-200,
// ISS-196). A genuine read failure is never folded into "not found".
func VerifyInput(repo lottery.Repository, input string) (*VerifyReport, error) {
	record, err := resolveRecord(repo, input)
	if err != nil {
		return nil, err
	}
	return VerifyRecord(record), nil
}

// resolveRecord turns a user reference into exactly one stored draw.
//
// The repository contract has two legal spellings for "no such row": the
// lottery.ErrNotFound sentinel (the SQLite implementation) and a nil record
// with a nil error (in-memory fakes, and the oracle module's documented
// convention). Both mean "keep resolving", so a nil record falls through to
// the height and substring steps instead of reaching VerifyRecord, which
// dereferences it. Only a genuine read failure stops resolution.
func resolveRecord(repo lottery.Repository, input string) (*lottery.LotteryRecord, error) {
	record, err := repo.GetByID(input)
	if err == nil && record != nil {
		return record, nil
	}
	if err != nil && !errors.Is(err, lottery.ErrNotFound) {
		return nil, fmt.Errorf("failed to read lottery record: %w", err)
	}

	if height, perr := strconv.ParseInt(input, 10, 64); perr == nil {
		records, rerr := repo.GetByBlockHeight(height)
		if rerr != nil {
			return nil, fmt.Errorf("failed to read by block height: %w", rerr)
		}
		for _, r := range records {
			if r != nil {
				return r, nil
			}
		}
		return nil, fmt.Errorf("lottery not found: %s", input)
	}

	all, gerr := repo.GetAll()
	if gerr != nil {
		return nil, fmt.Errorf("failed to read history: %w", gerr)
	}
	for _, r := range all {
		if r == nil {
			continue
		}
		if strings.Contains(r.ID, input) || strings.Contains(r.Seed, input) {
			return r, nil
		}
	}
	return nil, fmt.Errorf("lottery not found: %s", input)
}

// VerifyRecord checks a stored draw's integrity.
//
// Draws created before v1.31 have no persisted public key, so the check falls
// back to re-running SelectWinners on the stored VRF output and comparing the
// resulting set with the recorded winners. Draws from v1.31 onward also carry a
// VRFPublicKey; when present the cryptographic proof is checked as well. Every
// winner must remain paired with its deterministic NameToAddress value.
func VerifyRecord(record *lottery.LotteryRecord) *VerifyReport {
	report := &VerifyReport{Record: record}
	fail := func(format string, args ...any) {
		report.Checks = append(report.Checks, VerifyCheck{Status: VerifyFail, Message: fmt.Sprintf(format, args...)})
	}

	vrfOutputBytes, err := hex.DecodeString(record.VRFOutput)
	switch {
	case err != nil:
		fail("VRF output is not valid hex")
	case !isHex(record.VRFProof):
		fail("VRF proof is not valid hex")
	case record.ID != lottery.LotteryRecordID(record.Seed, vrfOutputBytes):
		fail("record ID does not match the seed and VRF output")
	case len(record.Winners) == 0:
		// A successful draw always records at least one winner (the domain
		// rejects count<=0), so an empty winners slice is a corrupt or partial
		// write. Without this, SelectWinners(output, roster, 0) returns an
		// empty set that vacuously "matches" (TASK-249, ISS-249).
		fail("record has no winners (corrupt or partial write)")
	default:
		expected := lottery.SelectWinners(vrfOutputBytes, record.Participants, len(record.Winners))
		if !sameStringSet(expected, record.Winners) {
			fail("stored winners do not match the VRF output (expected %v, stored %v)", expected, record.Winners)
			break
		}
		if err := verifyWinnerAddresses(record); err != nil {
			fail("%v", err)
			break
		}
		if record.VRFPublicKey == "" {
			report.Checks = append(report.Checks, VerifyCheck{
				Status:  VerifyOK,
				Message: "deterministic winner match (pre-v1.31 draw has no stored key)",
			})
			break
		}
		pk, keyErr := lottery.DecodePublicKey(record.VRFPublicKey)
		if keyErr != nil {
			fail("stored VRF public key is malformed")
			break
		}
		ok, _ := lottery.NewService().VerifyDraw(record, pk)
		if !ok {
			fail("VRF proof does not verify against the stored public key")
			break
		}
		report.ProofVerified = true
		report.Checks = append(report.Checks, VerifyCheck{
			Status:  VerifyOK,
			Message: "VRF proof verified against the stored public key",
		})
	}

	report.IntegrityOK = true
	for _, check := range report.Checks {
		if check.Status == VerifyFail {
			report.IntegrityOK = false
			break
		}
	}
	return report
}

func verifyWinnerAddresses(record *lottery.LotteryRecord) error {
	if len(record.Winners) != len(record.WinnerAddresses) {
		return fmt.Errorf(
			"winner/address length mismatch (%d winners, %d addresses)",
			len(record.Winners), len(record.WinnerAddresses),
		)
	}
	for i, winner := range record.Winners {
		want := lottery.NameToAddress(winner)
		if record.WinnerAddresses[i] != want {
			return fmt.Errorf(
				"winner address %d does not match winner %q (expected %s, stored %s)",
				i, winner, want, record.WinnerAddresses[i],
			)
		}
	}
	return nil
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}
