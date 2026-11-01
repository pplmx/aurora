package i18n

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNoModuleKeepsTwoKeysForOnePlaceholder covers a narrower question than
// "do any two keys share a text": across the catalog the same word legitimately
// appears under a module key and a shared one (`token.owner` and
// `nft.owner` are both "Owner public key" because both modules have an owner).
//
// What is not legitimate is one module keeping two keys for the same
// placeholder: `voting.none` and `common.none` were both "(none)" and
// `migrate.none` was a third. Inside one prefix, two identical values mean one
// of the keys lost its reason to exist, and a translator keeps both in step for
// nothing.
func TestNoModuleKeepsTwoKeysForOnePlaceholder(t *testing.T) {
	byPrefix := map[string]map[string]string{}
	duplicates := []string{}
	for key, value := range englishMessages {
		// A scoped key (`record.action.transfer@nft`) is a per-module variant of
		// the bare one; English spells both the same, which is exactly what the
		// scoped entry is for — the Chinese catalogs differ.
		if strings.Contains(key, "@") {
			continue
		}
		prefix, _, ok := strings.Cut(key, ".")
		if !ok {
			continue
		}
		if byPrefix[prefix] == nil {
			byPrefix[prefix] = map[string]string{}
		}
		if first, seen := byPrefix[prefix][value]; seen {
			duplicates = append(duplicates, first+" and "+key+" both say "+value)
			continue
		}
		byPrefix[prefix][value] = key
	}
	assert.Emptyf(t, duplicates,
		"one module keeps two keys for the same text; share one key instead")
}
