package config

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DbConfig  DatabaseConfig   `yaml:"db_config"`
	GeoConfig GeoServiceConfig `yaml:"geo_config"`
	GoosePath string           `yaml:"goose_path"`
}

func LoadConfig(path string, logger *slog.Logger) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read file", "error", err)
		return Config{}, fmt.Errorf("failed to read file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error("yaml parse error", "error", err)
		return Config{}, fmt.Errorf("yaml parse error: %w", err)
	}

	return cfg, nil
}
