package postgresqlclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Oleja123/estate-agency/internal/infrastructure/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewClient(ctx context.Context, logger slog.Logger, cfg config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.DbConfig.Username, cfg.DbConfig.Password, cfg.DbConfig.Host, cfg.DbConfig.Port, cfg.DbConfig.Database)

	maxAttempts := cfg.DbConfig.MaxAttempts

	for ; maxAttempts > 0; maxAttempts -= 1 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.DbConfig.SecondsToConnect)*time.Second)

		pool, err := pgxpool.New(ctx, dsn)
		cancel()
		if err != nil {
			slog.Warn("попытка провалилась", "error", err)
			continue
		}
		return pool, nil
	}
	slog.Warn("не удалось подключиться к БД")
	return nil, errors.New("не удалось подключиться к базе данных")
}
