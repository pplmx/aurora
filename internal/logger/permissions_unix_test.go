//go:build !windows

package logger

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setUmask(t *testing.T, mask int) {
	t.Helper()
	previous := syscall.Umask(mask)
	t.Cleanup(func() { syscall.Umask(previous) })
}

func TestInit_FileLoggerUsesOwnerOnlyPermissions(t *testing.T) {
	setUmask(t, 0o022)
	logPath := filepath.Join(t.TempDir(), "logs")
	setLogConfig(t, "info", logPath)

	Init()

	dirInfo, err := os.Stat(logPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(filepath.Join(logPath, "aurora.log"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}

func TestInit_FileLoggerRepairsExistingLoosePermissions(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "logs")
	require.NoError(t, os.Mkdir(logPath, 0o755))
	require.NoError(t, os.Chmod(logPath, 0o755))

	logFile := filepath.Join(logPath, "aurora.log")
	require.NoError(t, os.WriteFile(logFile, []byte("existing log\n"), 0o644))
	require.NoError(t, os.Chmod(logFile, 0o644))

	setLogConfig(t, "info", logPath)
	Init()

	dirInfo, err := os.Stat(logPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(logFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}
