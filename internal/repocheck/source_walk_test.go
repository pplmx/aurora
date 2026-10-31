package repocheck

import (
	"os"
	"path/filepath"
	"strings"
)

// walkGoSource calls fn with the path and body of every non-test Go file under
// root. It exists so the i18n checks read source without each one repeating the
// same directory walk.
func walkGoSource(root string, fn func(path, body string)) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
		if readErr != nil {
			return readErr
		}
		fn(path, string(body))
		return nil
	})
}
