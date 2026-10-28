package nativeui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRenderersUseResolvedThemeColors keeps the high-contrast preference from
// stopping at the toolkit theme. The window draws most of its chrome, text and
// borders through explicit painters; resolving those colours from c.Theme()
// is what lets the OS preference replace the muted text, border and selection
// tones. The palette constants remain the single source inside theme.go.
func TestRenderersUseResolvedThemeColors(t *testing.T) {
	forbidden := map[string]bool{
		"ink":            true,
		"inkMuted":       true,
		"border":         true,
		"selection":      true,
		"surface":        true,
		"surfaceHover":   true,
		"surfacePressed": true,
		"night":          true,
		"focusRing":      true,
	}

	var violations []string
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == "theme.go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		require.NoError(t, err)
		file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if ok && forbidden[ident.Name] {
				violations = append(violations, fmt.Sprintf("%s: %s", name, ident.Name))
			}
			return true
		})
	}
	require.Empty(t, violations,
		"renderers must use c.Theme() for OS high-contrast colors")
}
