package backup

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// restrictBackupFile makes a database snapshot or its metadata owner-only.
// SQLite's VACUUM INTO creates the snapshot with the process umask, so the
// file itself was 0644 under the common 022 default even when the caller
// selected a private directory. Windows backup directories inherit the user's
// ACL and do not use these Unix mode bits.
func restrictBackupFile(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, 0o600)
}

// restrictBackupDir tightens the hidden restore staging directory, including
// directories left behind by older releases at 0755. A deliberate symlink is
// rejected so Chmod cannot retarget an unrelated directory.
func restrictBackupDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is a symlink", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, 0o700)
}

// mkdirAllDurable creates path and flushes each new directory entry in its
// parent. Syncing only the leaf directory persists its contents, not the leaf
// itself: a desktop backup creates a new timestamp directory, so a power loss
// could otherwise remove the entire successfully reported backup.
func mkdirAllDurable(path string, perm os.FileMode) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve directory path: %w", err)
	}
	absolute = filepath.Clean(absolute)

	var missing []string
	for current := absolute; ; {
		info, statErr := os.Stat(current)
		if statErr == nil {
			if !info.IsDir() {
				return fmt.Errorf("%q is not a directory", current)
			}
			break
		}
		if !os.IsNotExist(statErr) {
			return fmt.Errorf("inspect directory %q: %w", current, statErr)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	if err := os.MkdirAll(absolute, perm); err != nil {
		return err
	}
	for i := len(missing) - 1; i >= 0; i-- {
		parent := filepath.Dir(missing[i])
		if err := syncDirFn(parent); err != nil {
			return fmt.Errorf("sync parent directory for %q: %w", missing[i], err)
		}
	}
	return nil
}

func copyFileAtomic(srcPath, dstPath string) error {
	_, err := copyFileAtomicWithHash(srcPath, dstPath)
	return err
}

// copyFileAtomicWithHash returns the SHA-256 of the bytes actually written.
// Restore compares it with the hash observed during verification so a source
// swap between those steps cannot bless different database contents.
func copyFileAtomicWithHash(srcPath, dstPath string) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		_ = src.Close()
		return "", fmt.Errorf("remove stale temp: %w", err)
	}
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		_ = src.Close()
		return "", fmt.Errorf("create temp: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, hash), src); err != nil {
		_ = src.Close()
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("copy: %w", err)
	}
	if err := src.Close(); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("close source: %w", err)
	}
	if err := syncFileFn(dst); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("sync temp: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("close temp: %w", err)
	}
	if err := restrictBackupFile(dstPath); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("restrict temp: %w", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func syncFileAtPath(path string) error {
	// Unix lets fsync operate on a read-only descriptor. Opening the file for
	// writing would reject a hardened 0400 backup even though the containing
	// directory permits replacing its entry. Windows' FlushFileBuffers needs a
	// write handle, so retain O_RDWR there.
	flags := os.O_RDONLY
	if runtime.GOOS == "windows" {
		flags = os.O_RDWR
	}
	f, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return err
	}
	if err := syncFileFn(f); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
