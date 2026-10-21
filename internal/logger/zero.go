package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pplmx/aurora/internal/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Log is the current logger. It stays a package variable for the tests and
// packages that install their own logger (middleware's request-id test swaps
// in a buffer), but it is only written under logMu: Init publishes while the
// oracle scheduler, the audit-outbox drainer and request handlers are already
// logging, and an unguarded assignment was a data race the detector reported.
//
// Every accessor below takes the lock, which also covers zerolog's global
// TimeFieldFormat that Init sets and every timestamped event reads.
var Log zerolog.Logger

// logMu guards Log.
var logMu sync.RWMutex

// logFile tracks the open file backing a file-based logger so tests (and any
// caller that re-Init()s) can release it. Without this the handle is only
// reachable through zerolog's internal writer; on Windows an open log file
// blocks deletion of the temp dir, and repeated Init() calls leak handles.
var logFile *os.File

// initMu serializes Init and Close: both touch logFile and the published
// logger, and a desktop start or a test calling Init again must not interleave
// with a background goroutine's logging.
var initMu sync.Mutex

type Config struct {
	LogLevel string
	LogPath  string
}

func Init() {
	initMu.Lock()
	defer initMu.Unlock()

	cfg := loadConfig()

	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	var output zerolog.Logger

	if cfg.LogPath != "" && cfg.LogPath != "./log" {
		// Try to use file
		//
		// Create the directory BEFORE normalizing it. NormalizePath requires the
		// target to exist, so validating first made every first run fall back to
		// console - including this repository's own shipped default
		// (config/aurora.toml sets log.path = "./logs/"), whose directory is
		// only created by the very call that never ran. A desktop app has no
		// console to fall back to, so that lost the whole log.
		if err := os.MkdirAll(cfg.LogPath, 0755); err != nil {
			fallbackToConsole(fmt.Sprintf("Failed to create log directory: %v", err))
		} else if normPath, err := utils.NormalizePath(cfg.LogPath); err != nil {
			fallbackToConsole(fmt.Sprintf("Failed to normalize log path: %v", err))
		} else {
			f, err := os.OpenFile(normPath+"/aurora.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fallbackToConsole(fmt.Sprintf("Failed to open log file: %v", err))
			} else {
				// Close any prior handle before replacing it (repeated
				// Init() calls used to leak the previous file). Called as
				// the locked body: Close takes initMu, which Init holds.
				closeFile()
				logFile = f
				output = zerolog.New(f).With().Timestamp().Logger()
				publish(output)
				return
			}
		}
	}

	// Default to console
	output = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	publish(output)
}

// publish installs l as the logger.
func publish(l zerolog.Logger) {
	logMu.Lock()
	Log = l
	logMu.Unlock()
}

// Close releases the file handle underlying a file-based logger. Tests use it
// in their cleanup so t.TempDir() can remove the log dir on Windows (an open
// file cannot be deleted there; Unix tolerates unlink-of-open-file).
func Close() {
	initMu.Lock()
	defer initMu.Unlock()
	closeFile()
}

// closeFile is Close's body, for callers that already hold initMu.
func closeFile() {
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}

// fallbackToConsole writes a warning message to stderr when file-based logging
// fails to initialize. This ensures the user is always aware that their
// configured log path is not being used.
func fallbackToConsole(msg string) {
	fmt.Fprintf(os.Stderr, "logger: falling back to console output: %s\n", msg)
}

func loadConfig() *Config {
	return &Config{
		LogLevel: viper.GetString("log.level"),
		LogPath:  viper.GetString("log.path"),
	}
}

func Info() *zerolog.Event {
	return withLogger((*zerolog.Logger).Info)
}

func Debug() *zerolog.Event {
	return withLogger((*zerolog.Logger).Debug)
}

func Error() *zerolog.Event {
	return withLogger((*zerolog.Logger).Error)
}

func Warn() *zerolog.Event {
	return withLogger((*zerolog.Logger).Warn)
}

func Fatal() *zerolog.Event {
	return withLogger((*zerolog.Logger).Fatal)
}

func With() zerolog.Context {
	return withLogger((*zerolog.Logger).With)
}

// withLogger calls fn on the current logger under the read lock, so an
// accessor can never read Log while Init is replacing it. zerolog builds the
// event immediately, so the lock is released before the caller writes it.
func withLogger[T any](fn func(*zerolog.Logger) T) T {
	logMu.RLock()
	defer logMu.RUnlock()
	return fn(&Log)
}

// Printf prints a formatted message at info level
func Printf(format string, v ...interface{}) {
	Info().Msg(fmt.Sprintf(format, v...))
}
