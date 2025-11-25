package utils

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunGooseMigrations runs migrations against the local test database using the goose library.
func RunGooseMigrations(logger *slog.Logger) error {
	migrationsPath, err := getMigrationsPath(logger)
	if err != nil {
		return fmt.Errorf("failed to get migrations path: %w", err)
	}

	logger.Info("running migrations", "path", migrationsPath)

	dsn := "postgres://root:root@localhost:5432/test?sslmode=disable"

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open DB connection: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error("failed to set goose dialect", "error", err)
	}

	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("failed to run migrations (goose): %w", err)
	}

	logger.Info("migrations applied successfully")
	return nil
}

func getMigrationsPath(logger *slog.Logger) (string, error) {
	possiblePaths := []string{
		"/home/oleg/estate-agency/db/migrations",
	}

	for _, path := range possiblePaths {
		absPath, _ := filepath.Abs(path)
		if _, err := os.Stat(absPath); err == nil {
			logger.Info("migrations folder found", "path", absPath)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("migrations folder not found: %v", possiblePaths)
}
