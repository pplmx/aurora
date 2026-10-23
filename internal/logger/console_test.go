package logger

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureProcessOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout, os.Stderr = stdoutWriter, stderrWriter
	defer func() { os.Stdout, os.Stderr = oldStdout, oldStderr }()

	fn()

	os.Stdout, os.Stderr = oldStdout, oldStderr
	require.NoError(t, stdoutWriter.Close())
	require.NoError(t, stderrWriter.Close())
	stdout, err := io.ReadAll(stdoutReader)
	require.NoError(t, err)
	stderr, err := io.ReadAll(stderrReader)
	require.NoError(t, err)
	require.NoError(t, stdoutReader.Close())
	require.NoError(t, stderrReader.Close())
	return string(stdout), string(stderr)
}

func TestInit_ConsoleFallbackWritesToStderr(t *testing.T) {
	setLogConfig(t, "info", "")

	stdout, stderr := captureProcessOutput(t, func() {
		Init()
		Info().Msg("console-fallback-marker")
	})

	assert.Empty(t, stdout, "structured logs must never corrupt machine-readable command output")
	assert.Contains(t, stderr, "console-fallback-marker")
}
