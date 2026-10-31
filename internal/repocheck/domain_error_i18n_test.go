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
	domainErrorLiteral = regexp.MustCompile(`errors\.New\("((?:[^"\\]|\\.)*)"\)`)
	oracleErrorLiteral = regexp.MustCompile(`OracleError\{Message: "((?:[^"\\]|\\.)*)"`)
	desktopZhEntry     = regexp.MustCompile(`(?m)^\t"([^"]+)":\s+"([^"]*)",$`)
)

// TestDomainErrorsAreTranslatedForTheDesktop keeps the two halves of an error
// message together. The domain reports in English, and the desktop looks the
// whole sentence up in its Chinese table before showing it; a message with no
// entry reaches a Chinese window in English while every surrounding word is
// Chinese. The oracle errors were exactly that: Fetch now and Delete showed
// "rate limit exceeded" and "source not found" untranslated.
func TestDomainErrorsAreTranslatedForTheDesktop(t *testing.T) {
	// Messages that never reach the window: they describe a programming fault,
	// or the desktop rewrites them into a module-specific sentence first.
	internal := map[string]string{
		"invalid event payload": "an internal decoding fault, not an operator condition",
		"record is nil":         "a programming fault in a repository call",
		"record not found":      "the desktop replaces this with a module-specific sentence",
	}

	messages := map[string]string{}
	root := filepath.Join(repoRoot(t), "internal", "domain")
	err := walkGoSource(root, func(path, body string) {
		for _, m := range domainErrorLiteral.FindAllStringSubmatch(body, -1) {
			messages[m[1]] = path
		}
		for _, m := range oracleErrorLiteral.FindAllStringSubmatch(body, -1) {
			messages[m[1]] = path
		}
	})
	require.NoError(t, err)
	require.Greater(t, len(messages), 50, "the domain declares its error messages")

	translations := map[string]string{}
	for _, m := range desktopZhEntry.FindAllStringSubmatch(readRepoFile(t, "desktop", "internal", "nativeui", "messages_zh.go"), -1) {
		translations[m[1]] = m[2]
	}
	require.NotEmpty(t, translations, "the desktop keeps a Chinese table")

	var missing []string
	for message := range messages {
		if _, ok := internal[message]; ok {
			continue
		}
		if _, ok := translations[message]; !ok {
			missing = append(missing, message)
		}
	}
	sort.Strings(missing)
	assert.Empty(t, missing, "these domain messages would reach a Chinese window in English")
}
