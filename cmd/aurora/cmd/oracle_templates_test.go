package cmd

import (
	"testing"

	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetTemplates_Sorted is a regression test for a non-deterministic
// output bug: the catalog used to be iterated directly without sorting, so
// `aurora oracle template list` order was randomized per Go map iteration.
// After the fix (shared app-layer catalog), output is lexicographically
// sorted by ID — same order across runs.
func TestGetTemplates_Sorted(t *testing.T) {
	templates := oracleapp.ListTemplates()
	ids := make([]string, 0, len(templates))
	for _, tpl := range templates {
		ids = append(ids, tpl.ID)
	}

	// Must contain the built-ins.
	require.Contains(t, ids, "btc-price")
	require.Contains(t, ids, "eth-price")

	// Must be sorted.
	for i := 1; i < len(ids); i++ {
		if ids[i-1] > ids[i] {
			t.Errorf("templates not sorted at index %d: %q > %q",
				i, ids[i-1], ids[i])
		}
	}
}

// TestGetTemplates_DeterministicOrder asserts the helper returns
// the same order across multiple invocations. Pre-fix, two calls
// could (and often did) disagree because of map randomization.
func TestGetTemplates_DeterministicOrder(t *testing.T) {
	first := oracleapp.ListTemplates()
	for i := 0; i < 10; i++ {
		got := oracleapp.ListTemplates()
		assert.Equal(t, first, got,
			"ListTemplates must return the same order on every call (got %#v vs %#v)",
			first, got)
	}
}

// TestGetTemplate_Found covers the happy path: looking up a
// built-in template returns the matching entry.
func TestGetTemplate_Found(t *testing.T) {
	ds, ok := oracleapp.GetTemplate("btc-price")
	require.True(t, ok, "btc-price should be a known template")
	assert.Equal(t, "Bitcoin Price", ds.Name)
	assert.Contains(t, ds.URL, "coingecko.com")
}

// TestGetTemplate_NotFound covers the missing case: callers must
// be able to tell when the name doesn't match anything so they
// can surface a clean error instead of dereferencing an empty
// struct.
func TestGetTemplate_NotFound(t *testing.T) {
	_, ok := oracleapp.GetTemplate("definitely-not-a-template")
	assert.False(t, ok, "unknown template name must return ok=false")
}
