package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testProfile = `mode: atomic
github.com/pplmx/aurora/internal/domain/nft/entity.go:10.1,12.2 2 1
github.com/pplmx/aurora/internal/domain/nft/entity.go:20.1,22.2 2 0
github.com/pplmx/aurora/internal/app/nftapp/usecase.go:5.1,6.2 1 1
github.com/pplmx/aurora/internal/api/handler.go:1.1,2.2 5 1
`

// TestParseProfileSumsStatementsPerPackage: a package's percentage is its
// covered statements over its total, not an average of block percentages.
func TestParseProfileSumsStatementsPerPackage(t *testing.T) {
	packages, err := parseProfile(testProfile)
	require.NoError(t, err)

	nft := packages["github.com/pplmx/aurora/internal/domain/nft"]
	require.NotNil(t, nft)
	require.Equal(t, 4, nft.total)
	require.Equal(t, 2, nft.covered)
	require.InDelta(t, 50, nft.percent(), 0.01)

	app := packages["github.com/pplmx/aurora/internal/app/nftapp"]
	require.Equal(t, 1, app.total)
	require.Equal(t, 1, app.covered)
	require.InDelta(t, 100, app.percent(), 0.01)
}

// TestParseProfileRejectsMalformedLines: a truncated profile must fail loudly
// instead of reporting 100% for an empty set.
func TestParseProfileRejectsMalformedLines(t *testing.T) {
	_, err := parseProfile("mode: atomic\ngarbage\n")
	require.Error(t, err)
}

// TestMatchesAnySelectsTheCheckedModules: only domain and app packages are
// subject to the floor; infra and the API layer have their own baselines.
func TestMatchesAnySelectsTheCheckedModules(t *testing.T) {
	prefixes := []string{
		"github.com/pplmx/aurora/internal/domain/",
		"github.com/pplmx/aurora/internal/app/",
	}
	require.True(t, matchesAny("github.com/pplmx/aurora/internal/domain/nft", prefixes))
	require.True(t, matchesAny("github.com/pplmx/aurora/internal/app/votingapp", prefixes))
	require.False(t, matchesAny("github.com/pplmx/aurora/internal/api", prefixes))
	require.False(t, matchesAny("github.com/pplmx/aurora/internal/infra/sqlite", prefixes))
}

// TestSplitPrefixesDropsEmptyEntries: `-exclude ""` must mean "exclude
// nothing"; an empty prefix would otherwise match every package.
func TestSplitPrefixesDropsEmptyEntries(t *testing.T) {
	require.Empty(t, splitPrefixes(""))
	require.Empty(t, splitPrefixes(" , "))
	require.Equal(t, []string{"a/", "b/"}, splitPrefixes("a/,, b/"))
}
