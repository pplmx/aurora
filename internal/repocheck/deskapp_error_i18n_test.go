package repocheck

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	deskappErrorLiteral = regexp.MustCompile(`(?:fmt\.Errorf|errors\.New)\("((?:[^"\\]|\\.)*)"`)
	deskappZhEntry      = regexp.MustCompile(`(?m)^\t"([^"]+)":\s+"([^"]*)",$`)
)

// TestDeskappErrorsAreTranslatedForTheWindow covers the messages the desktop
// adapter writes itself. Unlike a domain error, which at least names a rule, one
// of these is the whole explanation an operator gets: "blockchain not
// initialised" on a ledger card, "no data recorded for this source yet" under
// Get Latest, "desktop settings path is empty" beside the save that failed.
func TestDeskappErrorsAreTranslatedForTheWindow(t *testing.T) {
	translations := map[string]string{}
	for _, m := range deskappZhEntry.FindAllStringSubmatch(readRepoFile(t, "desktop", "internal", "nativeui", "messages_zh.go"), -1) {
		translations[m[1]] = m[2]
	}
	require.NotEmpty(t, translations, "the desktop keeps a Chinese table")

	missingSet := map[string]bool{}
	root := filepath.Join(repoRoot(t), "desktop", "internal", "deskapp")
	err := walkGoSource(root, func(path string, body string) {
		for _, m := range deskappErrorLiteral.FindAllStringSubmatch(body, -1) {
			message := m[1]
			// A message with a verb phrase is assembled at the call site and
			// translated by the localizer's head rule; only the fixed sentences
			// are looked up whole.
			if strings.Contains(message, "%") {
				continue
			}
			if _, ok := translations[message]; !ok {
				missingSet[message] = true
			}
		}
	})
	require.NoError(t, err)
	missing := make([]string, 0, len(missingSet))
	for message := range missingSet {
		missing = append(missing, message)
	}
	sort.Strings(missing)
	assert.Empty(t, missing, "these adapter messages would reach a Chinese window in English")
}
