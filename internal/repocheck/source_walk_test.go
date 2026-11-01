package repocheck

import (
	"errors"
	"io/fs"
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
			return walkErrToleratingDisappearance(err)
		}
		if d.IsDir() {
			if skipWalkDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
		if readErr != nil {
			return walkErrToleratingDisappearance(readErr)
		}
		fn(path, string(body))
		return nil
	})
}

// walkErrToleratingDisappearance turns "this entry vanished" into "nothing to
// read". The repocheck tests walk the checkout while the rest of the suite runs
// in other packages, and cmd/aurora/cmd's tests chdir into temp dirs and call
// store.ResetForTest, which removes ./data. A scan that failed on the resulting
// ENOENT reported a missing repository directory that never existed.
func walkErrToleratingDisappearance(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// skipWalkDir lists the directories a repository walk must not enter. The build
// and module caches hold tens of thousands of files — .gocache alone is 25 GB in
// a working checkout — and walking them made one translation test take twelve
// seconds. Every walk in this package shares the list so a new one cannot forget
// part of it. `.rumdl_cache` is a tool cache that carries its own .gitignore; it
// is small today but grows with the prose it indexes.
func skipWalkDir(name string) bool {
	switch name {
	case ".git", ".gocache", ".gomodcache", ".gopath", ".gotmp", ".rumdl_cache",
		"node_modules", "dist", ".screens", "logs", "data", ".planning", "testdata":
		return true
	}
	return false
}
