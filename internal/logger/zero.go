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
		if normPath, err := utils.NormalizePath(cfg.LogPath); err == nil {
			// Ensure directory exists
			if err := os.MkdirAll(normPath, 0755); err == nil {
				logFile, err := os.OpenFile(normPath+"/aurora.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					output = zerolog.New(logFile).With().Timestamp().Logger()
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
