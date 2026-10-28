package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// renameFile is a variable so tests can inject a failed promotion and prove
// the previous document survives it.
var renameFile = os.Rename

// syncDirectory makes the rename durable. Windows cannot fsync a directory
// handle through os.File, so its rename durability remains delegated to the
// platform; every Unix release flushes the containing directory.
var syncDirectory = func(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := dir.Sync()
	closeErr := dir.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

// WriteFileAtomic writes data to path through a temporary sibling and an
// atomic rename. Unlike os.WriteFile, a failed write or rename never leaves a
// truncated file at path: the previous document is either replaced by the
// complete new one or left untouched.
//
// An existing file keeps its permission bits. A symlink at path is resolved
// first, including a dangling link, so its target is updated instead of the
// link being replaced.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	resolved, err := ResolveFinalSymlink(path)
	if err != nil {
		return fmt.Errorf("resolve symlink %s: %w", path, err)
	}
	path = resolved

	mode := perm
	existing := false
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", path)
		}
		mode = info.Mode().Perm() | info.Mode()&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky)
		existing = true
	}

	tmp, err := createTempSibling(path, mode)
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	discard := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	// An existing file must keep its exact bits; a new file keeps the
	// caller's requested mode subject to the umask.
	if existing {
		if err := tmp.Chmod(mode); err != nil {
			discard()
			return fmt.Errorf("set mode on %s: %w", tmpName, err)
		}
	}
	if _, err := tmp.Write(data); err != nil {
		discard()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		discard()
		return fmt.Errorf("sync %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := renameFile(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("replace %s: %w", path, err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("sync directory for %s: %w", path, err)
	}
	return nil
}

// createTempSibling creates an exclusive temporary file next to path, so the
// later rename stays on one filesystem and remains atomic.
func createTempSibling(path string, perm fs.FileMode) (*os.File, error) {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	for range 100 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return nil, err
		}
		name := filepath.Join(dir, "."+base+"."+hex.EncodeToString(suffix[:])+".tmp")
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, errors.New("cannot create a unique temporary file next to " + base)
}
