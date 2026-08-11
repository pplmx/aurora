package cmd

import "testing"

// TestSameStringSet covers the helper used by verifyCmd to compare the
// re-derived winners against the stored winners.
//
// SelectWinners returns winners in the order they fall out of the VRF
// stream. If a record is exported/imported through a tool that reorders
// the array, an ordered comparison would falsely report tampering. We
// compare as multisets.
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
