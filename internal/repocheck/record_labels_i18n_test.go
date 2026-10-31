package repocheck

import (
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	recordFactLabel = regexp.MustCompile(`appendFact\([^,]+, "([A-Za-z ]+)"`)
	desktopCatalog  = regexp.MustCompile(`(?m)^\t"([^"]+)":`)
)

// TestRecordLabelsAreTranslated covers the one place the ledger's vocabulary
// lives outside the view package. The decoder names the facts an auditor reads
// — "Holder", "Spender", "Winners" — and the ledger card looks each label up in
// the desktop catalog, so a label the catalog lacks reaches a Chinese window in
// English. That is exactly what "Owner" did after the NFT page renamed it to
// "Holder": the page said 持有者 and the ledger row beside it said Owner.
func TestRecordLabelsAreTranslated(t *testing.T) {
	labels := map[string]bool{}
	err := walkGoSource(filepath.Join(repoRoot(t), "internal", "records"), func(_ string, body string) {
		for _, m := range recordFactLabel.FindAllStringSubmatch(body, -1) {
			labels[m[1]] = true
		}
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(labels), 10, "the decoder names its facts")

	catalogs := map[string]map[string]bool{}
	for _, name := range []string{"messages_zh.go", "messages_zh_hant.go"} {
		entries := map[string]bool{}
		for _, m := range desktopCatalog.FindAllStringSubmatch(readRepoFile(t, "desktop", "internal", "nativeui", name), -1) {
			entries[m[1]] = true
		}
		require.NotEmpty(t, entries, "%s must define labels", name)
		catalogs[name] = entries
	}

	names := make([]string, 0, len(labels))
	for label := range labels {
		names = append(names, label)
	}
	sort.Strings(names)

	for _, name := range []string{"messages_zh.go", "messages_zh_hant.go"} {
		for _, label := range names {
			assert.Truef(t, catalogs[name][label],
				"%s has no entry for the ledger label %q, so a ledger row shows it in English", name, label)
		}
	}
}
