package nativeui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The window's spacing and radius values live in design.go so a card, a button
// and a page header agree on rhythm instead of each file inventing its own
// number. Those constants are only worth having if the view code uses them: a
// literal radius that happens to equal a constant is invisible drift, and it is
// how the palette's panel, a key cap and the brand mark each ended up with a
// private corner radius that only coincidentally matched.
//
// This test reads the source rather than the rendered pixels because the
// failure is "a literal slipped in", which a screenshot cannot distinguish from
// a deliberate value.

// radiusLiteralsAllowed lists the numeric radii that are genuinely about a
// shape rather than about the design scale: a pill (999), a hairline bar
// (1.5), a bar track (3) and a square (0). Anything else must come from
// design.go.
var radiusLiteralsAllowed = map[string]bool{
	"0":   true, // a square segment, such as a tab
	"1.5": true, // the height of a 3px bar's rounded end
	"3":   true, // a 6px progress track's radius
	"999": true, // a pill
	"2":   true, // a 1px rule's rounded end
}

// TestRadiiComeFromTheDesignScale keeps corner radii on the shared scale. It
// walks every view file except design.go (which defines the scale) and reports
// a Radius call whose argument is a bare number.
func TestRadiiComeFromTheDesignScale(t *testing.T) {
	var violations []string

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") || name == "design.go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Radius" || len(call.Args) != 1 {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.INT && literal.Kind != token.FLOAT {
				return true
			}
			if radiusLiteralsAllowed[literal.Value] {
				return true
			}
			violations = append(violations,
				name+": Radius("+literal.Value+") should use a design.go constant")
			return true
		})
	}
	require.Empty(t, violations,
		"corner radii must come from the design scale so a card, a button and the chrome stay in step")
}

// TestSpacingComesFromTheDesignScale applies the same rule to padding. The
// design scale is a four-pixel rhythm; a literal padding is a number that will
// not agree with its neighbours when any of them changes.
func TestSpacingComesFromTheDesignScale(t *testing.T) {
	// Paddings that are about a single control's own geometry rather than the
	// page rhythm: the radius bar's inset, and a 2px nudge on a small chip.
	allowed := map[string]bool{"0": true, "1": true, "2": true, "5": true}

	var violations []string
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") || name == "design.go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Padding" {
				return true
			}
			for _, arg := range call.Args {
				literal, ok := arg.(*ast.BasicLit)
				if !ok || literal.Kind != token.INT {
					continue
				}
				if allowed[literal.Value] {
					continue
				}
				violations = append(violations,
					name+": Padding("+literal.Value+") should use a design.go constant")
			}
			return true
		})
	}
	require.Empty(t, violations,
		"padding must come from the design scale so cards keep a shared rhythm")
}
