package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDesktopErrorKeysNameARealError covers the desktop catalog's other role.
// Besides the labels the view passes to tr(), it holds translations for the
// English strings a service returns, matched by exact text through
// localizeServiceErrorMessage. A key that no code produces is dead weight that
// reads as coverage: "blockchain not initialised" sat in the catalog while the
// code said "initialized", so the window printed English and nothing failed.
func TestDesktopErrorKeysNameARealError(t *testing.T) {
	entry := regexp.MustCompile(`(?m)^\t"((?:[^"\\]|\\.)*)":\s+"((?:[^"\\]|\\.)*)",$`)
	view := strings.Join(goSourcesUnder(t, "desktop", "internal", "nativeui"), "\n")
	catalogs := []string{"messages_zh.go", "messages_zh_hant.go"}

	core := strings.Join(goSourcesUnder(t, "internal"), "\n")
	desktop := strings.Join(goSourcesUnder(t, "desktop", "internal"), "\n")

	checked, unattached := 0, []string{}
	for _, name := range catalogs {
		body := readRepoFile(t, "desktop", "internal", "nativeui", name)
		for _, pair := range entry.FindAllStringSubmatch(body, -1) {
			key := pair[1]
			// A label the view names verbatim is reachable by construction.
			if strings.Contains(view, key) {
				continue
			}
			checked++
			// The key must be recoverable from what a Go file writes. Three
			// shapes appear: the whole sentence as a literal, a wrapper that
			// appends a fragment (`%w: invalid interval`), and a prefix built by
			// concatenation (`"parse document: " + detail`). The last two mean
			// the key only ever appears in pieces, so the check looks for the
			// longest piece that still identifies it.
			if !emitsError(core, key) && !emitsError(desktop, key) {
				unattached = append(unattached, name+": "+key)
			}
		}
	}
	require.Greater(t, checked, 60, "the scan must reach the service-error keys")
	assert.Emptyf(t, unattached, "desktop translations no Go file produces: %v", unattached)
}

// goSourcesUnder returns the bodies of every non-test Go file below dir, except
// the catalogs themselves: a translation names its own key, so leaving them in
// makes every entry look like a call site.
func goSourcesUnder(t *testing.T, dir ...string) []string {
	t.Helper()
	root := filepath.Join(append([]string{repoRoot(t)}, dir...)...)
	bodies := []string{}
	require.NoError(t, walkGoSource(root, func(path, body string) {
		if strings.HasPrefix(filepath.Base(path), "messages_") {
			return
		}
		bodies = append(bodies, body)
	}))
	return bodies
}

// emitsError reports whether source can produce the English text the desktop
// table translates. It accepts the whole sentence, a leading fragment of it
// (a wrapper that appends the rest), or a trailing fragment (a concatenated
// prefix). A fragment shorter than eight bytes is ignored as too weak to mean
// anything.
func emitsError(source, key string) bool {
	if strings.Contains(source, `"`+key) {
		return true
	}
	for _, part := range strings.Split(key, ": ") {
		if len(part) >= 8 && strings.Contains(source, part) {
			return true
		}
	}
	for _, part := range strings.Split(key, ", got ") {
		if len(part) >= 8 && strings.Contains(source, part) {
			return true
		}
	}
	return false
}
