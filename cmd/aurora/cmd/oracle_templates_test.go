package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetTemplates_Sorted is a regression test for a non-deterministic
// output bug: getTemplates() used to iterate DataSourceTemplates
// directly without sorting, so `aurora oracle template list` output
// order was randomized per Go map iteration. After the fix, output
// is lexicographically sorted — same order across runs.
func TestGetTemplates_Sorted(t *testing.T) {
	templates := getTemplates()

	// Must contain the built-ins.
	require.Contains(t, templates, "btc-price")
	require.Contains(t, templates, "eth-price")

	// Must be sorted.
	for i := 1; i < len(templates); i++ {
		if templates[i-1] > templates[i] {
			t.Errorf("templates not sorted at index %d: %q > %q",
				i, templates[i-1], templates[i])
		}
	}
}

// TestGetTemplates_DeterministicOrder asserts the helper returns
// the same order across multiple invocations. Pre-fix, two calls
// could (and often did) disagree because of map randomization.
func TestGetTemplates_DeterministicOrder(t *testing.T) {
	first := getTemplates()
	for i := 0; i < 10; i++ {
		got := getTemplates()
		assert.Equal(t, first, got,
			"getTemplates must return the same order on every call (got %#v vs %#v)",
			first, got)
	}
}

// TestGetTemplate_Found covers the happy path: looking up a
// built-in template returns the matching entry.
func TestGetTemplate_Found(t *testing.T) {
	ds, ok := getTemplate("btc-price")
	require.True(t, ok, "btc-price should be a known template")
	assert.Equal(t, "Bitcoin Price", ds.Name)
	assert.Contains(t, ds.URL, "coingecko.com")
}

// TestGetTemplate_NotFound covers the missing case: callers must
// be able to tell when the name doesn't match anything so they
// can surface a clean error instead of dereferencing an empty
// struct.
func TestGetTemplate_NotFound(t *testing.T) {
	_, ok := getTemplate("definitely-not-a-template")
	assert.False(t, ok, "unknown template name must return ok=false")
}
