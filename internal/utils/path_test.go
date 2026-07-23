package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	type setupFn func(t *testing.T) (path, expectedAbs string)
	type assertFn func(t *testing.T, err error, expectedAbs string)

	tests := []struct {
		name    string
		setup   setupFn
		assert  assertFn
		wantErr bool
	}{
		{
			name: "absolute path to existing file",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				f := filepath.Join(t.TempDir(), "a.txt")
				require.NoError(t, os.WriteFile(f, []byte("hi"), 0o600))
				return f, f
			},
			assert: func(t *testing.T, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "relative path is resolved against cwd",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "r.txt"), nil, 0o600))

				cwd, err := os.Getwd()
				require.NoError(t, err)
				rel, err := filepath.Rel(cwd, filepath.Join(dir, "r.txt"))
				require.NoError(t, err)

				expected, err := filepath.Abs(rel)
				require.NoError(t, err)
				return rel, expected
			},
			assert: func(t *testing.T, err error, expected string) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "path with redundant separators is cleaned",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				dir := t.TempDir()
				f := filepath.Join(dir, "c.txt")
				require.NoError(t, os.WriteFile(f, nil, 0o600))
				dirty := dir + string(filepath.Separator) + string(filepath.Separator) + "." + string(filepath.Separator) + "c.txt"
				return dirty, f
			},
			assert: func(t *testing.T, err error, expected string) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "non-existent file returns wrapped os.Stat error",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				return filepath.Join(t.TempDir(), "does-not-exist"), ""
			},
			assert: func(t *testing.T, err error, _ string) {
				t.Helper()
				require.Error(t, err)
				assert.True(t, errors.Is(err, os.ErrNotExist),
					"expected os.ErrNotExist via errors.Is, got: %v", err)
				assert.Contains(t, err.Error(), "failed to get file info",
					"error should be wrapped with descriptive prefix")
			},
			wantErr: true,
		},
		{
			name: "empty path resolves to cwd",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				cwd, err := os.Getwd()
				require.NoError(t, err)
				return "", cwd
			},
			assert: func(t *testing.T, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "directory path is accepted",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				dir := t.TempDir()
				return dir, dir
			},
			assert: func(t *testing.T, err error, _ string) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, expectedAbs := tt.setup(t)

			got, err := NormalizePath(input)

			if tt.wantErr {
				assert.Error(t, err)
				tt.assert(t, err, expectedAbs)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, expectedAbs, got,
				"NormalizePath should return an absolute, cleaned path")
			assert.True(t, filepath.IsAbs(got),
				"result must be absolute, got %q", got)
		})
	}
}

func TestNormalizePath_ErrorMessageIncludesOriginalPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	_, err := NormalizePath(missing)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "missing.txt"),
		"error message should mention the offending path, got: %v", err)
}
