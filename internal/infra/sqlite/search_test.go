package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// likeContains is the one place the search escaping lives, so it is worth
// pinning on its own: seven repositories route through it now, and a change here
// reaches every collection at once.
func TestLikeContainsEscapesTheWildcards(t *testing.T) {
	cases := []struct {
		name string
		term string
		want string
	}{
		{"plain", "gold", "%gold%"},
		{"lowered", "GoLd", "%gold%"},
		{"trimmed", "  gold  ", "%gold%"},
		// % and _ are LIKE wildcards. A symbol or seed containing one must match
		// itself, or the search returns rows it has nothing to do with.
		{"percent", "100%", `%100\%%`},
		{"underscore", "a_b", `%a\_b%`},
		// The escape character itself has to survive first, or a term like \%\
		// would end up meaning something other than the three characters.
		{"backslash", `a\b`, `%a\\b%`},
		{"all three", `\%_`, `%\\\%\_%`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := likeContains(tc.term)
			require.True(t, ok, "a term with content must produce a pattern")
			assert.Equal(t, tc.want, got)
		})
	}

	for _, blank := range []string{"", "   ", "\t\n"} {
		_, ok := likeContains(blank)
		assert.Falsef(t, ok, "a blank term %q must not produce a pattern", blank)
	}
}
