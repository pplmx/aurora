package voting

import (
	"errors"
	"testing"
)

// TestValidateSessionTransition locks TASK-096 / ISS-088: the lifecycle's one
// hard rule is that an ENDED session can never be re-activated (a reopened
// ballot would silently accept votes again after EndSession closed it). Every
// other transition is idempotent and allowed.
func TestValidateSessionTransition(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want error
	}{
		{"draft to active", StatusDraft, StatusActive, nil},
		{"active to active (idempotent start)", StatusActive, StatusActive, nil},
		{"empty to active (legacy sessions)", "", StatusActive, nil},
		{"ended to active (REOPEN - forbidden)", StatusEnded, StatusActive, ErrSessionAlreadyEnded},
		{"draft to ended (cancel)", StatusDraft, StatusEnded, nil},
		{"active to ended (close)", StatusActive, StatusEnded, nil},
		{"ended to ended (idempotent end)", StatusEnded, StatusEnded, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateSessionTransition(tc.from, tc.to)
			if !errors.Is(got, tc.want) {
				t.Fatalf("ValidateSessionTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}
