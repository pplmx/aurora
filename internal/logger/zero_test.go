package logger

import (
	"testing"
)

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
