package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pplmx/aurora/internal/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

var Log zerolog.Logger

// logFile tracks the open file backing a file-based logger so tests (and any
// caller that re-Init()s) can release it. Without this the handle is only
// reachable through zerolog's internal writer; on Windows an open log file
// blocks deletion of the temp dir, and repeated Init() calls leak handles.
var logFile *os.File

type Config struct {
	LogLevel string
	LogPath  string
}

func Init() {
	cfg := loadConfig()

	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure output
	zerolog.TimeFieldFormat = time.RFC3339

	var output zerolog.Logger

	if cfg.LogPath != "" && cfg.LogPath != "./log" {
		// Try to use file
		if normPath, err := utils.NormalizePath(cfg.LogPath); err != nil {
			fallbackToConsole(fmt.Sprintf("Failed to normalize log path: %v", err))
		} else {
			// Ensure directory exists
			if err := os.MkdirAll(normPath, 0755); err != nil {
				fallbackToConsole(fmt.Sprintf("Failed to create log directory: %v", err))
			} else {
				f, err := os.OpenFile(normPath+"/aurora.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fallbackToConsole(fmt.Sprintf("Failed to open log file: %v", err))
				} else {
					// Close any prior handle before replacing it (repeated
					// Init() calls used to leak the previous file).
					Close()
					logFile = f
					output = zerolog.New(f).With().Timestamp().Logger()
					Log = output
					return
				}
			}
		}
	}

	// Default to console
	output = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	Log = output
}

// Close releases the file handle underlying a file-based logger. Tests use it
// in their cleanup so t.TempDir() can remove the log dir on Windows (an open
// file cannot be deleted there; Unix tolerates unlink-of-open-file).
func Close() {
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
	return Log.Info()
}

func Debug() *zerolog.Event {
	return Log.Debug()
}

func Error() *zerolog.Event {
	return Log.Error()
}

func Warn() *zerolog.Event {
	return Log.Warn()
}

func Fatal() *zerolog.Event {
	return Log.Fatal()
}

func With() zerolog.Context {
	return Log.With()
}

// Printf prints a formatted message at info level
func Printf(format string, v ...interface{}) {
	Log.Info().Msg(fmt.Sprintf(format, v...))
}
