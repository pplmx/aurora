package ledger

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrityReportAlwaysCarriesTheBrokenIndex covers the field a caller
// branches on. It carried `omitempty`, so a break at index 0 serialized without
// the field entirely:
//
//	{"valid":false,"length":3,"first_broken_reason":"nil block"}
//
// Index 0 is reachable through the nil-chain path, which reports the failure at
// the chain's first block, and through any future caller that verifies a slice
// whose head is broken. A client reading the documented contract ("-1 means
// intact, otherwise the index of the first break") would decode 0 and conclude
// the chain was fine, which is the state it also uses for "intact".
func TestIntegrityReportAlwaysCarriesTheBrokenIndex(t *testing.T) {
	cases := []struct {
		name   string
		report IntegrityReport
		want   int
	}{
		{
			name:   "break at the first block",
			report: IntegrityReport{Valid: false, Length: 3, FirstBrokenIndex: 0, FirstBrokenReason: "nil chain"},
			want:   0,
		},
		{
			name:   "break in the middle",
			report: IntegrityReport{Valid: false, Length: 3, FirstBrokenIndex: 1, FirstBrokenReason: "prev-hash chain break"},
			want:   1,
		},
		{
			name:   "intact chain",
			report: IntegrityReport{Valid: true, Length: 3, FirstBrokenIndex: -1, Head: "ab"},
			want:   -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := json.Marshal(tc.report)
			require.NoError(t, err)

			var decoded map[string]any
			require.NoError(t, json.Unmarshal(out, &decoded))
			index, ok := decoded["first_broken_index"]
			require.Truef(t, ok, "the report must always carry the index: %s", out)
			assert.EqualValues(t, tc.want, index, "json: %s", out)
		})
	}
}
