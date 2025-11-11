package config

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DbConfig  DatabaseConfig
	GeoConfig GeoServiceConfig
}

func LoadConfig(path string, logger *slog.Logger) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("ошибка чтения файла", "error", err)
		return Config{}, fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error("ошибка парсинга yaml", "error", err)
		return Config{}, fmt.Errorf("ошибка парсинга yaml: %w", err)
	}

	return cfg, nil
}
