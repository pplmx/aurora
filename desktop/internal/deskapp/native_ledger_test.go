package deskapp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/records"
)

// TestLedgerStatusReportsTheChainHead pins the three numbers the Blockchain
// page leads with. Height is derived as Len()-1, which is only meaningful
// because a chain always carries its genesis block — the adapter relies on that
// invariant, so the test records it rather than leaving it implicit.
func TestLedgerStatusReportsTheChainHead(t *testing.T) {
	chain := blockchain.New()
	services := &nativeServices{chain: chain}

	fresh, err := services.LedgerStatus()
	require.NoError(t, err)
	require.True(t, fresh.Valid, "a chain with only its genesis block is intact")
	require.Equal(t, 1, fresh.Blocks, "genesis counts as a stored block")
	require.Equal(t, 0, fresh.Height, "the genesis block is height 0, not -1")
	require.NotEmpty(t, fresh.Head, "the page shows the head hash as the chain's identity")

	_, err = chain.AddBlock(`{"module":"lottery"}`)
	require.NoError(t, err)
	_, err = chain.AddBlock(`{"module":"voting"}`)
	require.NoError(t, err)

	grown, err := services.LedgerStatus()
	require.NoError(t, err)
	require.True(t, grown.Valid)
	require.Equal(t, 3, grown.Blocks)
	require.Equal(t, 2, grown.Height, "two records on top of genesis")
	require.NotEqual(t, fresh.Head, grown.Head, "the head moves as blocks are appended")
}

// TestLedgerStatusWithoutAChainIsAnError keeps the unwired case visible: a
// status of zero values would render as a verified ledger with no blocks.
func TestLedgerStatusWithoutAChainIsAnError(t *testing.T) {
	services := &nativeServices{}

	_, err := services.LedgerStatus()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not initialized")
}

// TestLedgerStatusReportsABrokenChain covers the state the page exists to
// surface: a tampered block makes the report invalid instead of quietly
// looking like any other chain.
func TestLedgerStatusReportsABrokenChain(t *testing.T) {
	chain := blockchain.New()
	_, err := chain.AddBlock(`{"module":"lottery"}`)
	require.NoError(t, err)
	// Rewriting a stored payload keeps the hash it was mined with, so the chain
	// no longer verifies.
	chain.Blocks[1].Data = []byte(`{"module":"tampered"}`)

	status, err := (&nativeServices{chain: chain}).LedgerStatus()
	require.NoError(t, err)
	require.False(t, status.Valid, "a tampered block must not read as verified")
}

// ledgerSearchFixture builds a services value whose chain holds one token record,
// so a search can be tried against a real chain rather than a fake that has to
// reimplement the matching it is meant to prove.
func ledgerSearchFixture(t *testing.T) (*nativeServices, *blockchain.Chain) {
	t.Helper()
	chain := blockchain.New()
	// The pipe shapes put the verb first: `mint|<symbol>|<to>|<amount>` is what
	// the token service writes, and records.Parse reads the module back from it.
	_, err := chain.AddBlock(`mint|GLD|alice|10`)
	require.NoError(t, err)
	return &nativeServices{chain: chain}, chain
}

// TestTheLedgerSearchUnderstandsTheWindowsLanguage checks the search term the
// backlog footer writes into the field. The chain stores module ids ("token"),
// but a Chinese window shows 代币 and a reader who sees that word expects it to
// find those records. The search text includes the localized vocabulary for
// exactly this reason; this pins it so a future simplification to "match the raw
// payload" cannot quietly drop the word the reader is looking at.
func TestTheLedgerSearchUnderstandsTheWindowsLanguage(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	for _, tc := range []struct{ locale, term string }{
		{"en", "token"},
		{"zh", nativeui.RecordVocabulary(records.KindModule, "token", "token")},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			tr.SetLocale(tc.locale)
			services, chain := ledgerSearchFixture(t)
			page, err := services.LedgerBlocksPage(tc.term, 10, 0)
			require.NoError(t, err)
			require.NotEmpty(t, page.Rows,
				"the term %q shown in a %s window must find the token record (chain height %d)",
				tc.term, tc.locale, chain.Len())
			require.Equal(t, "token", page.Rows[0].Module)
		})
	}
}
