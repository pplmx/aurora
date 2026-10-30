package repocheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// snakeCaseKey is the only shape an API JSON key may take: lowercase words
// joined by underscores. The handler package's per-response check only sees the
// types a test happens to serialize; this one walks the source, so a DTO that
// nothing exercises today still cannot ship a camelCase key.
var snakeCaseKey = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// TestJSONTagsAreSnakeCase scans the app DTOs and the API's own response types.
// It parses the files instead of reflecting over a registry because the set of
// response types is exactly "the structs with json tags under these packages".
func TestJSONTagsAreSnakeCase(t *testing.T) {
	root := repoRoot(t)
	scanned := 0
	for _, dir := range []string{"internal/app", "internal/api"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if parseErr != nil {
				return parseErr
			}
			rel, relErr := filepath.Rel(root, path)
			require.NoError(t, relErr)
			ast.Inspect(file, func(node ast.Node) bool {
				field, ok := node.(*ast.Field)
				if !ok || field.Tag == nil {
					return true
				}
				raw, unquoteErr := strconv.Unquote(field.Tag.Value)
				if unquoteErr != nil {
					return true
				}
				tag, present := reflect.StructTag(raw).Lookup("json")
				if !present {
					return true
				}
				name, _, _ := strings.Cut(tag, ",")
				if name == "" || name == "-" {
					return true
				}
				scanned++
				require.Regexpf(t, snakeCaseKey, name,
					"%s: json key %q is not snake_case", rel, name)
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}
	require.Greater(t, scanned, 50, "the scan must see the response DTOs")
}
