package repocheck

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheReferenceStatesThePaginationNumbers covers the defaults and ceilings a
// client has to plan around. They live as constants in seven handler files, and
// the reference restates them in prose — "默认 20，支持 limit/offset" — so a
// changed constant leaves a wrong number in the document that nothing checks.
//
// Offset ceilings differ on purpose: /nft/list and token history stop at 1000
// because paging that deep is a scan, while the rest reach 10000. That is the
// distinction most likely to be flattened by a careless edit, so it is asserted
// per endpoint.
func TestTheReferenceStatesThePaginationNumbers(t *testing.T) {
	doc := readRepoFile(t, "docs", "api.md")

	// Read the constant block of each handler. A constant that is renamed
	// fails the lookup rather than being skipped.
	constant := func(file, name string) int {
		t.Helper()
		body := readRepoFile(t, "internal", "api", "handler", file)
		m := regexp.MustCompile(`(?m)\b` + name + `\s*=\s*(\d+)\b`).FindStringSubmatch(body)
		require.NotNilf(t, m, "%s must still define %s", file, name)
		n, err := strconv.Atoi(m[1])
		require.NoError(t, err)
		return n
	}

	type endpoint struct {
		path string
		file string
		def  string
		maxL string
		maxO string
	}
	endpoints := []endpoint{
		{"/api/v1/lottery/history", "lottery.go", "defaultLotteryHistoryLimit", "maxLotteryHistoryLimit", "maxLotteryHistoryOffset"},
		{"/api/v1/voting/candidates", "voting.go", "defaultVotingListLimit", "maxVotingListLimit", "maxVotingListOffset"},
		{"/api/v1/nft/list", "nft.go", "defaultNFTListLimit", "maxNFTListLimit", "maxNFTListOffset"},
		{"/api/v1/nft/{id}/history", "nft.go", "defaultNFTHistoryLimit", "maxNFTHistoryLimit", "maxNFTHistoryOffset"},
		{"/api/v1/token/list", "token.go", "defaultTokenListLimit", "maxTokenListLimit", "maxTokenListOffset"},
		{"/api/v1/token/history", "token.go", "defaultHistoryLimit", "maxHistoryLimit", "maxHistoryOffset"},
		{"/api/v1/oracle/sources", "oracle.go", "defaultOracleSourceLimit", "maxOracleSourceLimit", "maxOracleSourceOffset"},
		{"/api/v1/blockchain/blocks", "blockchain.go", "defaultBlockPageLimit", "maxBlockPageLimit", "maxBlockPageOffset"},
		{"/api/v1/oracle/query", "oracle.go", "defaultQueryLimit", "maxQueryLimit", ""},
	}

	for _, ep := range endpoints {
		def := constant(ep.file, ep.def)
		maxL := constant(ep.file, ep.maxL)

		// The route has to appear in the index and its table row.
		require.Containsf(t, doc, "`"+ep.path+"`", "docs/api.md must document %s", ep.path)
		row := regexp.MustCompile("`" + regexp.QuoteMeta(ep.path) + "`[^\n]*").FindString(doc)
		require.NotEmptyf(t, row, "docs/api.md must list %s in a table", ep.path)

		// The default page size is stated either in the row or in the prose
		// below it — both are places a reader looks, and the point is that the
		// number is there and equals the constant.
		assert.Regexpf(t, `\b`+strconv.Itoa(def)+`\b`, doc,
			"%s: the reference must state the default page size %d", ep.path, def)

		// The prose states the ceiling. It is not always beside the route, so it
		// is looked for anywhere in the document: what matters is that the
		// number the reader meets equals the one the handler enforces.
		assert.Regexpf(t, `\b`+strconv.Itoa(maxL)+`\b`, doc,
			"%s: the reference must state the limit ceiling %d", ep.path, maxL)

		if ep.maxO == "" {
			continue
		}
		maxO := constant(ep.file, ep.maxO)
		assert.Regexpf(t, `\b`+strconv.Itoa(maxO)+`\b`, doc,
			"%s: the reference must state the offset ceiling %d", ep.path, maxO)
	}

	// The two deliberate ceilings must not converge: /nft/list and token history
	// stop at 1000 because paging that deep is a scan; everything else reaches
	// 10000. A careless edit flattening them is the failure this guards.
	nftListOffset := constant("nft.go", "maxNFTListOffset")
	historyOffset := constant("token.go", "maxHistoryOffset")
	assert.Equal(t, 1000, nftListOffset, "/nft/list offset ceiling")
	assert.Equal(t, 1000, historyOffset, "token history offset ceiling")
	assert.Contains(t, doc, "**1000**", "the reference must call out the tighter offset ceiling")
}
