package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveFinalSymlink follows the final component's symlink chain without
// requiring the destination to exist. Normal open/write semantics create the
// target of a dangling symlink, so callers that replace a file atomically must
// operate on that target instead of renaming over the link.
func ResolveFinalSymlink(path string) (string, error) {
	const maxSymlinkDepth = 255
	configuredPath := path
	for range maxSymlinkDepth {
		info, err := os.Lstat(path)
		switch {
		case os.IsNotExist(err):
			return path, nil
		case err != nil:
			return "", fmt.Errorf("inspect %q: %w", path, err)
		case info.Mode()&os.ModeSymlink == 0:
			return path, nil
		}

		target, err := os.Readlink(path)
		if err != nil {
			return "", fmt.Errorf("read symlink %q: %w", path, err)
		}
		if !filepath.IsAbs(target) {
			parent, err := filepath.EvalSymlinks(filepath.Dir(path))
			if err != nil {
				return "", fmt.Errorf("resolve parent of symlink %q: %w", path, err)
			}
			target = filepath.Join(parent, target)
		} else {
			target = filepath.Clean(target)
		}
		path = target
	}
	return "", fmt.Errorf("resolve %q: too many levels of symbolic links", configuredPath)
}
