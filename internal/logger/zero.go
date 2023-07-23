package logger

import (
	"os"
	"strings"

	"github.com/pplmx/aurora/internal/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type Config struct {
	LogLevel string // Log output level; Value: Debug, Info, Error; Default: Info
	LogPath  string // Log output path; Default: stdout
}

func loadConfig() *Config {
	logLevel := strings.ToLower(viper.GetString("log.level"))
	logPath := viper.GetString("log.path")
	return &Config{
		LogLevel: logLevel,
		LogPath:  logPath,
	}
}

func setupLogger() {
	cfg := loadConfig()

	// 1. configure log level
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// 2. configure log output
	// if not empty and is a valid path, write to file
	if cfg.LogPath != "" {
		// stdout as default,
		zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

		// check if the log path is valid
		if normPath, err := utils.NormalizePath(cfg.LogPath); err == nil {
			if logFile, err := os.OpenFile(normPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				zerolog.New(logFile).With().Timestamp().Logger()
				return
			}
		}
	}
}
