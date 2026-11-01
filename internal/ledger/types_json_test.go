package ledger

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrityReportAlwaysCarriesTheBrokenIndex covers the field a caller
// branches on. It carried `omitempty`, so a break at index 0 — the genesis
// block, which is where a corrupted or truncated database usually fails first —
// serialized without the field entirely:
//
//	{"valid":false,"length":3,"first_broken_reason":"nil block"}
//
// A client reading the documented contract ("-1 means intact, otherwise the
// index of the first break") would decode 0 and conclude the chain was fine, or
// report "unknown" for the one case it can name.
func TestIntegrityReportAlwaysCarriesTheBrokenIndex(t *testing.T) {
	cases := []struct {
		name   string
		report IntegrityReport
		want   int
	}{
		{
			name:   "break at the genesis block",
			report: IntegrityReport{Valid: false, Length: 3, FirstBrokenIndex: 0, FirstBrokenReason: "nil block"},
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
