package deskapp

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNativeAdapterUsesApplicationPorts keeps the view adapter on the domain
// interfaces. It may map domain entities and call app use cases, but must not
// reach storage or HTTP implementations directly.
func TestNativeAdapterUsesApplicationPorts(t *testing.T) {
	files, err := filepath.Glob("native_*.go")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imp := range file.Imports {
			value, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			for _, forbidden := range []string{"/internal/infra", "/internal/api"} {
				require.NotContainsf(t, value, forbidden,
					"%s imports %s; the adapter must use application/domain ports", path, value)
			}
		}
	}
}
