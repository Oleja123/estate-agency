package utils

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func RunGooseMigrations(logger *slog.Logger) error {
	migrationsPath, err := getMigrationsPath(logger)
	if err != nil {
		return fmt.Errorf("ошибка получения путей для миграций: %w", err)
	}

	logger.Info("запуск миграций", "путь", migrationsPath)

	cmd := exec.Command("goose",
		"-dir", migrationsPath,
		"postgres", "postgres://root:root@localhost:5432/test?sslmode=disable",
		"up",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка при запуске миграций: %w", err)
	}

	logger.Info("миграции успешно установлены")
	return nil
}

func getMigrationsPath(logger *slog.Logger) (string, error) {
	possiblePaths := []string{
		"/home/oleg/estate-agency/db/migrations",
	}

	for _, path := range possiblePaths {
		absPath, _ := filepath.Abs(path)
		if _, err := os.Stat(absPath); err == nil {
			logger.Info("нашлась папка миграций", "путь", absPath)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("папка миграций не найдена: %v", possiblePaths)
}
