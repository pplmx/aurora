package lotteryapp

import (
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

func verifiableLotteryRecord(t *testing.T) *lottery.LotteryRecord {
	t.Helper()
	participants := []string{"Alice", "Bob", "Carol"}
	service := lottery.NewService()
	winners, addresses, output, proof, publicKey, err := service.DrawWinners(
		participants, "address-integrity-seed", 2,
	)
	if err != nil {
		t.Fatalf("DrawWinners: %v", err)
	}
	return lottery.CreateLotteryRecord(
		"address-integrity-seed", participants, winners, addresses, output, proof, publicKey, 1,
	)
}

func TestVerifyRecord_AcceptsCanonicalWinnerAddresses(t *testing.T) {
	report := VerifyRecord(verifiableLotteryRecord(t))
	if !report.IntegrityOK {
		t.Fatalf("canonical draw must verify: %+v", report.Checks)
	}
	if !report.ProofVerified {
		t.Fatal("keyed draw must include proof verification")
	}
}

func TestVerifyRecord_RejectsTruncatedWinnerAddresses(t *testing.T) {
	record := verifiableLotteryRecord(t)
	record.WinnerAddresses = record.WinnerAddresses[:len(record.WinnerAddresses)-1]

	report := VerifyRecord(record)
	if report.IntegrityOK {
		t.Fatal("a missing winner address must fail verification")
	}
	if !hasVerifyFailure(report, "winner/address") {
		t.Fatalf("failure must identify the address mismatch: %+v", report.Checks)
	}
}

func TestVerifyRecord_RejectsTamperedWinnerAddress(t *testing.T) {
	record := verifiableLotteryRecord(t)
	record.WinnerAddresses[0] = lottery.NameToAddress("Mallory")

	report := VerifyRecord(record)
	if report.IntegrityOK {
		t.Fatal("a same-length tampered winner address must fail verification")
	}
	if !hasVerifyFailure(report, "winner address") {
		t.Fatalf("failure must identify the tampered address: %+v", report.Checks)
	}
}

func TestVerifyRecord_AcceptsConsistentlyReorderedWinnerAddresses(t *testing.T) {
	record := verifiableLotteryRecord(t)
	record.Winners[0], record.Winners[1] = record.Winners[1], record.Winners[0]
	record.WinnerAddresses[0], record.WinnerAddresses[1] =
		record.WinnerAddresses[1], record.WinnerAddresses[0]

	report := VerifyRecord(record)
	if !report.IntegrityOK {
		t.Fatalf("reordering winner/address pairs must remain verifiable: %+v", report.Checks)
	}
}

func TestVerifyRecord_RejectsRecordIDNotDerivedFromEvidence(t *testing.T) {
	record := verifiableLotteryRecord(t)
	record.ID = strings.Repeat("0", 16)

	report := VerifyRecord(record)
	if report.IntegrityOK {
		t.Fatal("a record ID not derived from the seed and VRF output must fail verification")
	}
	if !hasVerifyFailure(report, "record ID") {
		t.Fatalf("failure must identify the record ID mismatch: %+v", report.Checks)
	}
}

func hasVerifyFailure(report *VerifyReport, fragment string) bool {
	for _, check := range report.Checks {
		if check.Status == VerifyFail && strings.Contains(check.Message, fragment) {
			return true
		}
	}
	return false
}

// TestSameStringSet covers the helper that compares re-derived winners with
// the stored ones.
//
// SelectWinners returns winners in the order they fall out of the VRF stream.
// If a record is exported/imported through a tool that reorders the array, an
// ordered comparison would falsely report tampering, so the comparison is a
// multiset one.
func TestSameStringSet(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"nil and nil", nil, nil, true},
		{"identical", []string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{"same set, different order", []string{"a", "b", "c"}, []string{"c", "a", "b"}, true},
		{"duplicates match", []string{"a", "a", "b"}, []string{"a", "b", "a"}, true},
		{"different lengths", []string{"a", "b"}, []string{"a", "b", "c"}, false},
		{"different element", []string{"a", "b", "c"}, []string{"a", "b", "d"}, false},
		{"different multiplicity", []string{"a", "a", "b"}, []string{"a", "b", "b"}, false},
		{"one nil one empty", nil, []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sameStringSet(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("sameStringSet(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			// Symmetry: sameStringSet must be commutative.
			if sameStringSet(tt.b, tt.a) != tt.want {
				t.Errorf("sameStringSet(%v, %v) != sameStringSet(%v, %v) — not symmetric", tt.b, tt.a, tt.a, tt.b)
			}
		})
	}
}
