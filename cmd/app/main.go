package main

import (
	"context"
	"log/slog"
	"os"

	favoriteservice "github.com/Oleja123/estate-agency/internal/application/favorite"
	imageservice "github.com/Oleja123/estate-agency/internal/application/image"
	propertyservice "github.com/Oleja123/estate-agency/internal/application/property"
	propertytypeservice "github.com/Oleja123/estate-agency/internal/application/property_type"
	"github.com/Oleja123/estate-agency/internal/application/token"
	userservice "github.com/Oleja123/estate-agency/internal/application/user"
	"github.com/Oleja123/estate-agency/internal/application/user/password"
	postgresqlclient "github.com/Oleja123/estate-agency/internal/infrastructure/client/postgresql"
	"github.com/Oleja123/estate-agency/internal/infrastructure/config"
	favoritedb "github.com/Oleja123/estate-agency/internal/infrastructure/favorite"
	geocoder "github.com/Oleja123/estate-agency/internal/infrastructure/geocoder"
	propertydb "github.com/Oleja123/estate-agency/internal/infrastructure/property"
	propertytypedb "github.com/Oleja123/estate-agency/internal/infrastructure/property_type"
	userdb "github.com/Oleja123/estate-agency/internal/infrastructure/user"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(cfgPath, logger)
	if err != nil {
		logger.Error("failed to load config", "err", err, "path", cfgPath)
		os.Exit(1)
	}

	// Log non-sensitive config parts for visibility
	logger.Info("config loaded",
		"db_host", cfg.DbConfig.Host,
		"db_name", cfg.DbConfig.Database,
		"goose_path", cfg.GoosePath,
	)

	dbClient, err := postgresqlclient.NewClient(context.Background(), *logger, cfg)

	if err != nil {
		logger.Error("failed to create db client", "err", err)
		os.Exit(1)
	}

	userStorage := userdb.New(dbClient, logger)
	propertyStorage := propertydb.New(dbClient, logger)
	favoriteStorage := favoritedb.New(dbClient, logger)
	imageStorage := propertydb.NewImageRepository(dbClient, logger)
	propertyTypeStorage := propertytypedb.New(dbClient, logger)

	userService := userservice.New(userStorage, logger, password.NewBcryptHasher(),
		token.NewMemoryService())
	propertyTypeService := propertytypeservice.New(propertyTypeStorage, logger)
	propertyService := propertyservice.New(propertyStorage, propertyTypeStorage, logger, geocoder.NewNoop())
	favoriteService := favoriteservice.New(favoriteStorage, logger)
	imageService := imageservice.New(imageStorage, logger, "")

	// suppress unused variables until wiring is completed
	_ = userStorage
	_ = propertyStorage
	_ = favoriteStorage
	_ = imageStorage
	_ = propertyTypeStorage
	_ = userService
	_ = propertyTypeService
	_ = propertyService

}
