package repocheck

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryHandlerIsRegistered catches the one wiring mistake a compiler cannot:
// a handler method written, tested against a fake router, and never added to its
// module's Routes table. The route then 404s in production while the handler and
// its test both pass.
func TestEveryHandlerIsRegistered(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "internal", "api", "handler")
	handler := regexp.MustCompile(`func \(h \*\w+\) ([A-Z]\w*)\(w http\.ResponseWriter`)
	routesFn := regexp.MustCompile(`(?s)func \(h \*\w+\) Routes\(r chi\.Router\) \{(.*?)\n\}`)
	// The table passes method values: r.Post("/create", h.Create).
	reference := regexp.MustCompile(`\bh\.([A-Z]\w*)\)`)

	checked, unwired := 0, []string{}
	require.NoError(t, walkGoSource(dir, func(path, body string) {
		base := filepath.Base(path)
		if base == "helpers.go" || base == "page.go" {
			return
		}
		methods := handler.FindAllStringSubmatch(body, -1)
		if len(methods) == 0 {
			return
		}
		registered := map[string]bool{}
		for _, block := range routesFn.FindAllStringSubmatch(body, -1) {
			for _, m := range reference.FindAllStringSubmatch(block[1], -1) {
				registered[m[1]] = true
			}
		}
		for _, m := range methods {
			checked++
			if !registered[m[1]] {
				unwired = append(unwired, base+": "+m[1])
			}
		}
	}))
	require.Greater(t, checked, 35, "the scan must reach the handler methods")
	assert.Emptyf(t, unwired, "handler methods with no route: %v", unwired)
}
