package sqlite

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/token"
)

// The three collection searches added for the window share one contract: the
// filter runs over the stored columns, case-insensitively, and paging happens
// inside the match set. Filtering a loaded page cannot do that, which is the
// whole reason these reads exist.
func TestTokenListMatchingSearchesStoredText(t *testing.T) {
	repo, cleanup := setupTokenTestDB(t)
	defer cleanup()

	for _, tok := range []*token.Token{
		token.NewToken("GOLD", "Gold Standard", "AU", token.NewAmount(100), token.PublicKey([]byte("owner"))),
		token.NewToken("SILVER", "Silver Coin", "AG", token.NewAmount(200), token.PublicKey([]byte("owner"))),
		token.NewToken("BRONZE", "Bronze 100%", "BZ", token.NewAmount(300), token.PublicKey([]byte("owner"))),
		token.NewToken("IRON", "Iron 1000", "FE", token.NewAmount(400), token.PublicKey([]byte("owner"))),
	} {
		require.NoError(t, repo.SaveToken(tok))
	}

	byName, err := repo.ListMatchingTokens("gOlD", 0, 0)
	require.NoError(t, err)
	require.Len(t, byName, 1, "a name match is case-insensitive")
	assert.Equal(t, "GOLD", string(byName[0].ID()))

	bySymbol, err := repo.ListMatchingTokens("ag", 0, 0)
	require.NoError(t, err)
	require.Len(t, bySymbol, 1)
	assert.Equal(t, "SILVER", string(bySymbol[0].ID()))

	byID, err := repo.ListMatchingTokens("bronze", 0, 0)
	require.NoError(t, err)
	require.Len(t, byID, 1)

	// A percent sign is a LIKE wildcard; the search must treat it literally. The
	// "Iron 1000" row is the point: unescaped, "100%" matches it too, so a lenient
	// check here would pass with the escaping removed.
	literal, err := repo.ListMatchingTokens("100%", 0, 0)
	require.NoError(t, err)
	require.Len(t, literal, 1, "a percent sign must match itself, not any run of characters")
	assert.Equal(t, "BRONZE", string(literal[0].ID()))

	// Paging happens inside the match set, so page two of a match set is not
	// page two of the store.
	all, err := repo.ListMatchingTokens("o", 1, 1)
	require.NoError(t, err)
	require.Len(t, all, 1)

	empty, err := repo.ListMatchingTokens("   ", 0, 0)
	require.NoError(t, err)
	assert.Len(t, empty, 4, "a blank term is the plain list")
}

func TestNFTListMatchingSearchesStoredText(t *testing.T) {
	repo, cleanup := setupNFTTestDB(t)
	defer cleanup()

	// NewNFT leaves the id to the service, which assigns a uuid; the rows are
	// written straight here, so each needs its own or INSERT OR REPLACE would
	// leave one row behind.
	for i, name := range []string{"Star", "Moon", "Cloud_9", "CloudX9"} {
		n := nft.NewNFT(name, "", "", "", []byte("creator"), []byte("owner"))
		n.ID = "nft-" + strconv.Itoa(i)
		require.NoError(t, repo.SaveNFT(n))
	}

	byName, err := repo.ListMatchingNFTs("mOoN", 0, 0)
	require.NoError(t, err)
	require.Len(t, byName, 1)
	assert.Equal(t, "Moon", byName[0].Name)

	// An underscore is a LIKE wildcard too, so it must match literally. The
	// CloudX9 row is the point of this assertion: unescaped, "_9" matches it as
	// well, so a lenient check here would pass with the escaping removed.
	literal, err := repo.ListMatchingNFTs("_9", 0, 0)
	require.NoError(t, err)
	require.Len(t, literal, 1, "an underscore must match itself, not any character")
	assert.Equal(t, "Cloud_9", literal[0].Name)

	blank, err := repo.ListMatchingNFTs("", 0, 0)
	require.NoError(t, err)
	assert.Len(t, blank, 4)
}

func TestOracleListMatchingSearchesStoredText(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	for _, ds := range []*oracle.DataSource{
		{ID: "src-json", Name: "Price feed", URL: "https://example.com/price", Type: "json", Enabled: true},
		{ID: "src-xml", Name: "Weather feed", URL: "https://example.com/weather", Type: "xml", Enabled: true},
		{ID: "src-100pct", Name: "Percent feed", URL: "https://example.com/100%", Type: "json", Enabled: true},
		{ID: "src-1000", Name: "Thousand feed", URL: "https://example.com/1000", Type: "json", Enabled: true},
	} {
		require.NoError(t, repo.SaveSource(ds))
	}

	byType, err := repo.ListMatchingSources("XML", 0, 0)
	require.NoError(t, err)
	require.Len(t, byType, 1)
	assert.Equal(t, "src-xml", byType[0].ID, "the type column is searchable")

	byURL, err := repo.ListMatchingSources("weather", 0, 0)
	require.NoError(t, err)
	require.Len(t, byURL, 1)
	assert.Equal(t, "src-xml", byURL[0].ID, "so is the URL")

	// The "1000" feed is what makes this real: unescaped, "%" matches it too.
	literal, err := repo.ListMatchingSources("100%", 0, 0)
	require.NoError(t, err)
	require.Len(t, literal, 1, "a percent sign must match itself, not any run of characters")
	assert.Equal(t, "src-100pct", literal[0].ID)

	blank, err := repo.ListMatchingSources("  ", 0, 0)
	require.NoError(t, err)
	assert.Len(t, blank, 4)
}
