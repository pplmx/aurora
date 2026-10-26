package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Get is the entry point every surface uses — the CLI, the API
// server, the oracle scheduler and the desktop all take the chain from it —
// and it had no test. Its contract is a process-wide singleton: the second
// caller must get the same chain, because a second chain over the same
// database would mine on a stale height and fork the ledger.

// withCleanSingleton runs fn with the package singleton reset, so the test does
// not inherit a chain another test built.
func withCleanSingleton(t *testing.T, fn func()) {
	t.Helper()
	ResetForTest()
	t.Cleanup(ResetForTest)
	fn()
}

// TestGet_ReturnsTheSameInstance pins the singleton: two calls return
// one chain, and it is the one Init built.
func TestGet_ReturnsTheSameInstance(t *testing.T) {
	withCleanSingleton(t, func() {
		first := Get()
		require.NotNil(t, first)
		assert.Same(t, first, Get(),
			"every caller must share one chain")
		assert.Same(t, first, Init(),
			"an explicit init must return the chain already built")
	})
}

// TestGet_InitialisesLazily pins the first-call path: a nil singleton
// is built on demand rather than returning nil to a caller that then derefs it.
func TestGet_InitialisesLazily(t *testing.T) {
	withCleanSingleton(t, func() {
		// The reset above leaves the singleton nil; the call must build it.
		chain := Get()
		require.NotNil(t, chain, "the first caller must get a chain, not nil")
		require.NotEmpty(t, chain.Blocks, "a fresh chain carries genesis")
		assert.Equal(t, int64(0), chain.Blocks[0].Height)
	})
}

// TestGet_AppendsAreVisibleToEveryCaller proves the shared chain is
// actually shared: a block added through one reference is what the next
// Get call sees. Two chains over one database is the failure this
// prevents — each would mine on its own stale height.
func TestGet_AppendsAreVisibleToEveryCaller(t *testing.T) {
	withCleanSingleton(t, func() {
		height, err := Get().AddBlock("shared-payload")
		require.NoError(t, err)
		require.Equal(t, int64(1), height)

		assert.Equal(t, 1, Get().Len()-1,
			"a block added through one reference must be visible through the next call")
		assert.Equal(t, 1, Init().Len()-1,
			"and through an explicit init")
	})
}
