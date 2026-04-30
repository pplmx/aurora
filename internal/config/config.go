package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var (
	ErrMissingAPIKey  = errors.New("AURORA_API_KEY environment variable is required in production")
	ErrInsecureAPIKey = errors.New("insecure API key detected; please set a secure AURORA_API_KEY")
)

var insecureKeys = map[string]bool{
	"aurora-api-key-default": true,
	"changeme":               true,
	"secret":                 true,
	"api-key":                true,
	"":                       true,
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
	DB     DBConfig     `mapstructure:"db"`
	API    APIConfig    `mapstructure:"api"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

type DBConfig struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
}

type APIConfig struct {
	Key string `mapstructure:"key"`
}

func Load() (*Config, error) {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "./logs")
	viper.SetDefault("db.type", "sqlite")
	viper.SetDefault("db.path", "./data/aurora.db")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := validateAPIKey(cfg.API.Key); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func GetAPIKey() string {
	return viper.GetString("api.key")
}

func GenerateAPIKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate API key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(key), nil
}

func validateAPIKey(key string) error {
	isProduction := strings.ToLower(os.Getenv("AURORA_ENV")) == "production"

	if isProduction {
		if key == "" {
			return ErrMissingAPIKey
		}
		if insecureKeys[key] {
			return ErrInsecureAPIKey
		}
	} else if key == "" {
		generated, err := GenerateAPIKey()
		if err != nil {
			return fmt.Errorf("generate development API key: %w", err)
		}
		fmt.Printf("Generated development API key: %s\n", generated)
	}

	return nil
}
