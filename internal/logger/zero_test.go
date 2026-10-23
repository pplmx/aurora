package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// setLogConfig installs a temp log config (and restores viper afterwards) so
// Init() exercises the requested path.
func setLogConfig(t *testing.T, level, path string) {
	t.Helper()
	viper.Set("log.level", level)
	viper.Set("log.path", path)
	t.Cleanup(viper.Reset)
	// Close the file logger handle before t.TempDir()'s cleanup removes the
	// log dir (Windows cannot delete an open file).
	t.Cleanup(Close)
}

func TestLogger_Init(t *testing.T) {
	Init()
}

func TestLogger_Info(t *testing.T) {
	Init()
	Info().Msg("test")
}

func TestLogger_Debug(t *testing.T) {
	Init()
	Debug().Msg("test")
}

func TestLogger_Error(t *testing.T) {
	Init()
	Error().Msg("test")
}

func TestLogger_Warn(t *testing.T) {
	Init()
	Warn().Msg("test")
}

func TestLogger_With(t *testing.T) {
	Init()
	_ = With().Str("key", "value").Logger()
}

func TestLogger_Printf(t *testing.T) {
	Init()

	Printf("test message: %s", "hello")

	Printf("number: %d", 42)
}

func TestConfig_Default(t *testing.T) {
	Init()
	cfg := loadConfig()

	if cfg == nil {
		t.Error("Config should not be nil")
	}
}

// TestInit_FileLogger drives the file-output branch: a valid, existing log
// directory makes Init wire a file logger, and a debug record lands in
// aurora.log (the message is visible only when the emitted level is enabled).
func TestInit_FileLogger(t *testing.T) {
	setLogConfig(t, "debug", t.TempDir())

	Init()

	Info().Msg("file-logger-marker")
	Debug().Msg("file-logger-debug")

	data, err := os.ReadFile(filepath.Join(viper.GetString("log.path"), "aurora.log"))
	require.NoError(t, err, "aurora.log should be created in the configured path")
	require.Contains(t, string(data), "file-logger-marker")
	require.Contains(t, string(data), "file-logger-debug")
}

// TestInit_FileLoggerCreatesMissingDirectory is the first-run defect: Init
// validated the log directory with NormalizePath (which requires it to exist)
// before creating it, so a fresh checkout - and this repository's own
// config/aurora.toml default `log.path = "./logs/"` - fell back to the console
// logger and wrote no file at all. A packaged desktop app has no console, so
// the entire log was lost.
func TestInit_FileLoggerCreatesMissingDirectory(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "no-such-dir", "logs")
	setLogConfig(t, "info", logPath)

	Init()
	Info().Msg("first-run-marker")

	require.DirExists(t, logPath, "Init must create the configured log directory")
	data, err := os.ReadFile(filepath.Join(logPath, "aurora.log"))
	require.NoError(t, err, "the file logger must be used, not the console fallback")
	require.Contains(t, string(data), "first-run-marker")
}

// TestInit_FileLoggerFallback_MkdirFailure covers the fallback path where the
// normalized log path exists but is a regular file: MkdirAll fails and Init
// falls back to the console logger rather than panicking.
func TestInit_FileLoggerFallback_MkdirFailure(t *testing.T) {
	file := filepath.Join(t.TempDir(), "logfile")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	setLogConfig(t, "info", file)
	Init()
	require.NotNil(t, Log)
}

// TestInit_InvalidLevel covers the ParseLevel failure branch: an unknown level
// degrades to Info (global level is the only observable effect; the record
// above Info is suppressed).
func TestInit_InvalidLevel(t *testing.T) {
	setLogConfig(t, "not-a-level", t.TempDir())

	Init()
	require.NotNil(t, Log)
}

// TestInit_EmptyLevelDefaultsToInfo pins the CLI bootstrap path: before the
// config file is loaded, viper returns an empty log level. zerolog.ParseLevel
// treats "" as NoLevel without an error, which silently suppressed every
// ordinary log instead of using the documented info default.
func TestInit_EmptyLevelDefaultsToInfo(t *testing.T) {
	logPath := t.TempDir()
	setLogConfig(t, "", logPath)

	Init()
	Info().Msg("empty-level-marker")

	data, err := os.ReadFile(filepath.Join(logPath, "aurora.log"))
	require.NoError(t, err)
	require.Contains(t, string(data), "empty-level-marker")
}

// TestLogger_Fatal constructs a fatal event without sending it. zerolog only
// calls os.Exit when a Fatal event is sent (Msg), so merely obtaining the
// event must not terminate the test process.
func TestLogger_Fatal(t *testing.T) {
	setLogConfig(t, "info", t.TempDir())
	Init()

	ev := Fatal()
	require.NotNil(t, ev)
}

// TestLogger_DebugSuppressedByInfoLevel asserts the level filter actually
// works end-to-end: a debug record is dropped when the global level is info.
func TestLogger_DebugSuppressedByInfoLevel(t *testing.T) {
	setLogConfig(t, "info", t.TempDir())
	Init()

	Debug().Msg("should-be-suppressed")

	data, err := os.ReadFile(filepath.Join(viper.GetString("log.path"), "aurora.log"))
	require.NoError(t, err)
	require.False(t, strings.Contains(string(data), "should-be-suppressed"))
}
