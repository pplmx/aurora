package repocheck

import (
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	recordModuleWord = regexp.MustCompile(`Record\{Module: "([a-z]+)"`)
	recordActionWord = regexp.MustCompile(`(?m)^\tcase "[a-z_]+":\n\t\treturn "([A-Za-z ]+)"`)
	recordFactWord   = regexp.MustCompile(`appendFact\((?:nil|facts), "([A-Za-z ]+)"`)
	recordCatalogKey = regexp.MustCompile(`(?m)^\t"(record\.[a-z. @]+)":`)
)

// TestTheRecordVocabularyIsTranslated covers the words a decoded ledger record
// carries. The CLI resolves each one through a `record.<kind>.<word>` key, so a
// word with no key prints in English inside an otherwise Chinese row — which is
// exactly what every module name, every action and every fact label did before
// the rows were localized.
//
// The words are read from the decoder rather than listed here: a new module or a
// renamed fact must fail this check, not quietly fall back.
func TestTheRecordVocabularyIsTranslated(t *testing.T) {
	body := readRepoFile(t, "internal", "records", "records.go")

	kinds := map[string][]string{}
	for _, m := range recordModuleWord.FindAllStringSubmatch(body, -1) {
		kinds["module"] = append(kinds["module"], m[1])
	}
	for _, m := range recordActionWord.FindAllStringSubmatch(body, -1) {
		kinds["action"] = append(kinds["action"], m[1])
	}
	for _, m := range recordFactWord.FindAllStringSubmatch(body, -1) {
		kinds["fact"] = append(kinds["fact"], m[1])
	}
	for kind, words := range kinds {
		require.NotEmptyf(t, words, "the decoder emits no %s word; the scan must reach them", kind)
	}

	for _, name := range []string{"messages_en.go", "messages_zh.go", "messages_zh_hant.go"} {
		catalog := readRepoFile(t, "internal", "i18n", name)
		present := map[string]bool{}
		for _, m := range recordCatalogKey.FindAllStringSubmatch(catalog, -1) {
			present[m[1]] = true
		}
		require.NotEmptyf(t, present, "%s must define the record vocabulary", name)

		for _, kind := range []string{"module", "action", "fact"} {
			words := append([]string(nil), kinds[kind]...)
			sort.Strings(words)
			for _, word := range words {
				// A module-scoped entry (record.<kind>.<word>@<module>) is an
				// alternative spelling for one module, not a replacement: the
				// bare entry stays the fallback every other module uses.
				key := "record." + kind + "." + lowerASCII(word)
				assert.Truef(t, present[key],
					"%s has no %q, so a ledger row prints %q in English", name, key, word)
			}
		}
	}
}

// lowerASCII lowercases the ASCII letters of a decoder word. It is spelled out
// rather than using strings.ToLower so the key a caller builds
// (record.<kind>.<lowercased word>) and the key this check looks for cannot
// disagree about anything but case.
func lowerASCII(word string) string {
	out := []byte(word)
	for i, c := range out {
		if c >= 'A' && c <= 'Z' {
			out[i] = c + ('a' - 'A')
		}
	}
	return string(out)
}

// TestTheScopedRecordWordsAreReal covers the module-scoped entries: they are the
// way one English word gets two translations. An entry whose module or word no
// decoder path produces is unreachable text — record.action.transfer@nft exists
// because a token transfer is 转账 and an NFT transfer is 转让, and a typo in
// either half would silently fall back to the bare entry.
func TestTheScopedRecordWordsAreReal(t *testing.T) {
	body := readRepoFile(t, "internal", "records", "records.go")

	modules := map[string]bool{}
	for _, m := range recordModuleWord.FindAllStringSubmatch(body, -1) {
		modules[m[1]] = true
	}
	require.NotEmpty(t, modules, "the decoder names its modules")

	entries := regexp.MustCompile(`(?m)^\t"record\.([a-z]+)\.([a-z ]+)@([a-z]+)":`)
	checked := 0
	for _, name := range []string{"messages_en.go", "messages_zh.go", "messages_zh_hant.go"} {
		catalog := readRepoFile(t, "internal", "i18n", name)
		for _, m := range entries.FindAllStringSubmatch(catalog, -1) {
			kind, word, module := m[1], m[2], m[3]
			checked++
			assert.Truef(t, modules[module],
				"%s scopes %q to module %q, which the decoder never emits", name, word, module)
			// The bare entry must exist too: every other module falls back to it.
			bare := "record." + kind + "." + word
			assert.Containsf(t, catalog, `"`+bare+`":`,
				"%s scopes %q but has no bare %s for the other modules", name, word, bare)
		}
	}
	require.Greater(t, checked, 0, "the scan must reach a scoped entry")
}
