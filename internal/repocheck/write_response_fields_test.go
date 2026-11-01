package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheReferenceExplainsTheOneWriteWithoutABlockHeight covers a difference a
// client is likely to code against. Every token write response carries
// block_height except approve, because ApproveEvent has no block height — the
// domain records the event id, both public keys, the amount and a timestamp.
//
// A client that infers success from the presence of block_height therefore
// misreads an approval. The asymmetry is real and stays; the reference has to
// say so, since the shape table lists all five writes together.
func TestTheReferenceExplainsTheOneWriteWithoutABlockHeight(t *testing.T) {
	dto := readRepoFile(t, "internal", "app", "tokenapp", "dto.go")

	// The asymmetry must still exist: if approve ever gains the field, the note
	// below becomes wrong and this check sends the reader to fix it.
	approve := regexp.MustCompile(`(?s)type ApproveResponse struct \{(.*?)\n\}`).FindStringSubmatch(dto)
	require.NotNil(t, approve, "tokenapp must still define ApproveResponse")
	assert.NotContains(t, approve[1], "block_height",
		"approve has no block height; if that changed, update the reference's note")

	for _, name := range []string{"MintResponse", "TransferResponse", "TransferFromResponse", "BurnResponse"} {
		body := regexp.MustCompile(`(?s)type ` + name + ` struct \{(.*?)\n\}`).FindStringSubmatch(dto)
		require.NotNilf(t, body, "tokenapp must still define %s", name)
		assert.Containsf(t, body[1], "block_height",
			"%s carries block_height, which is what makes approve's absence worth explaining", name)
	}

	doc := readRepoFile(t, "docs", "api.md")
	require.Contains(t, doc, "**`approve` 不带**",
		"docs/api.md must say the approve response carries no block_height")
	assert.True(t, strings.Contains(doc, "/blockchain/blocks?q="),
		"docs/api.md must point at how to find the block an approval landed in")
}
