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

// skipWalkDir lists the directories a repository walk must not enter. The build
// and module caches hold tens of thousands of files — .gocache alone is 25 GB in
// a working checkout — and walking them made one translation test take twelve
// seconds. Every walk in this package shares the list so a new one cannot forget
// part of it.
func skipWalkDir(name string) bool {
	switch name {
	case ".git", ".gocache", ".gomodcache", ".gopath", ".gotmp",
		"node_modules", "dist", ".screens", "logs", "data", ".planning", "testdata":
		return true
	}
	return false
}
