package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReservedErrorCodesAreMarkedInTheReference covers two codes the classifier
// knows and no use case returns. The reference lists them among the errors a
// caller can receive, which invites a branch that never runs, and it is the
// kind of claim that goes stale silently: the day a use case starts returning
// one, the note must come out.
func TestReservedErrorCodesAreMarkedInTheReference(t *testing.T) {
	handler := readRepoFile(t, "internal", "api", "handler", "helpers.go")
	doc := readRepoFile(t, "docs", "api.md")

	start := indexOf(t, handler, "var errorClassification")
	pairs := regexp.MustCompile(`\{\s*([\w.]+)\s*,\s*http\.\w+\s*,\s*"([A-Z_]+)"\s*\}`).
		FindAllStringSubmatch(handler[start:], -1)
	require.Greater(t, len(pairs), 60, "the classifier must be parsed")

	sources := map[string]string{}
	require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "internal"), func(path, body string) {
		sources[path] = body
	}))
	reserved := []string{}
	for _, pair := range pairs {
		sentinel, code := pair[1], pair[2]
		name := sentinel[indexOf(t, sentinel, ".")+1:]
		// The domain file that defines the sentinel does not count as a use.
		used := false
		for path, body := range sources {
			if strings.HasSuffix(path, "domain/token/errors.go") || strings.HasSuffix(path, handlerPath) {
				continue
			}
			if regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(body) {
				used = true
				break
			}
		}
		if !used {
			reserved = append(reserved, code)
			assert.NotEqualf(t, -1, strings.Index(doc, "`"+code+"`"),
				"%s is defined but never returned; the reference must still list it", code)
		}
	}
	assert.ElementsMatch(t, []string{"NOT_TOKEN_OWNER", "DUPLICATE_TRANSFER"}, reserved,
		"a code with no producer must be named in the reference's reserved note")
	// Compared on the search result: a failing Contains prints the document.
	assert.NotEqualf(t, -1, strings.Index(doc, "当前不会产生"),
		"the reference must say which listed codes the server does not produce")
}

const handlerPath = "internal/api/handler/helpers.go"

func indexOf(t *testing.T, s, sub string) int {
	t.Helper()
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	t.Fatalf("%q not found", sub)
	return -1
}
