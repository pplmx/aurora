package config

import (
	"github.com/spf13/viper"
)

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
	viper.SetDefault("api.key", "aurora-api-key-default")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func GetAPIKey() string {
	return viper.GetString("api.key")
}
