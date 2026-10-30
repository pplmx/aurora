package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPageParams pins the shared paging contract: missing or unparsable values
// keep the defaults, a limit above its cap is clamped, and a negative offset is
// ignored rather than rejected.
func TestPageParams(t *testing.T) {
	cases := []struct {
		query  string
		limit  int
		offset int
	}{
		{"", 20, 0},
		{"?limit=5&offset=7", 5, 7},
		{"?limit=500", 100, 0},
		{"?offset=99999", 20, 10000},
		{"?limit=abc&offset=-1", 20, 0},
		{"?limit=0&offset=0", 20, 0},
		{"?limit=-5", 20, 0},
		{"?limit=1.5", 20, 0},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("GET", "/x"+tc.query, nil)
		limit, offset := pageParams(r, 20, 100, 10000)
		require.Equalf(t, tc.limit, limit, "limit for %q", tc.query)
		require.Equalf(t, tc.offset, offset, "offset for %q", tc.query)
	}
}

// TestPageLimit covers the endpoints that take a limit but no offset.
func TestPageLimit(t *testing.T) {
	for query, want := range map[string]int{
		"":            10,
		"?limit=3":    3,
		"?limit=999":  100,
		"?limit=zero": 10,
		"?limit=0":    10,
		"?offset=5":   10,
		"?limit=100":  100,
	} {
		r := httptest.NewRequest("GET", "/x"+query, nil)
		require.Equalf(t, want, pageLimit(r, 10, 100), "limit for %q", query)
	}
}
