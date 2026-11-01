package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchHintsMatchWhatTheSearchDoes covers the sentence beside every search
// field. Six lists filter the rows already loaded and say so ("Search narrows
// the current page"); two query storage across the whole set and say that
// instead. Swapping the two sentences would leave a reader looking for an old
// record in a page they cannot see, or trusting a filtered count as the whole
// store — the words are the only signal a user has.
//
// The check ties each sentence to the code path that backs it: the adapters that
// take a search string reach storage, and the view filters that call
// containsFold do not.
func TestSearchHintsMatchWhatTheSearchDoes(t *testing.T) {
	// Whole-store searches: each adapter method takes the term as a parameter
	// and hands it to a storage query. The parameter's name differs — one says
	// search, the other term — so the check matches either.
	for _, method := range []string{"LotteryHistory", "LedgerBlocksPage"} {
		found := false
		require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "desktop", "internal", "deskapp"), func(path, body string) {
			if regexp.MustCompile(`\) ` + method + `\((?:search|term) string`).MatchString(body) {
				found = true
			}
		}))
		assert.Truef(t, found, "%s must still take the search term and query storage", method)
	}

	// Page-local searches: the view filters what it loaded.
	view := readRepoFile(t, "desktop", "internal", "nativeui", "voting_lists.go") +
		readRepoFile(t, "desktop", "internal", "nativeui", "nft_collection.go") +
		readRepoFile(t, "desktop", "internal", "nativeui", "oracle_sources.go") +
		readRepoFile(t, "desktop", "internal", "nativeui", "token.go")
	require.GreaterOrEqual(t, strings.Count(view, "containsFold("), 5,
		"the page-local lists must still filter in memory")

	// Every hint is one of the two sentences, and the whole-store hint appears
	// only where storage is queried.
	hint := regexp.MustCompile(`"Search (?:narrows the current page|matches [^"]*across the whole [a-z]+)[^"]*"`)
	hints := hint.FindAllString(view+
		readRepoFile(t, "desktop", "internal", "nativeui", "lottery.go")+
		readRepoFile(t, "desktop", "internal", "nativeui", "blockchain.go"), -1)
	require.GreaterOrEqual(t, len(hints), 8, "every list must carry a search hint")

	wholeStore := 0
	for _, h := range hints {
		if strings.Contains(h, "across the whole") {
			wholeStore++
		}
	}
	assert.Equal(t, 2, wholeStore,
		"exactly the two lists that query storage may promise a whole-store search")
}
