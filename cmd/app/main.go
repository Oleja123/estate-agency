package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	favoriteservice "github.com/Oleja123/estate-agency/internal/application/favorite"
	imageservice "github.com/Oleja123/estate-agency/internal/application/image"
	propertyservice "github.com/Oleja123/estate-agency/internal/application/property"
	propertytypeservice "github.com/Oleja123/estate-agency/internal/application/property_type"
	"github.com/Oleja123/estate-agency/internal/application/token"
	userservice "github.com/Oleja123/estate-agency/internal/application/user"
	"github.com/Oleja123/estate-agency/internal/application/user/password"
	httpHandler "github.com/Oleja123/estate-agency/internal/handler"
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

	// token service shared between handlers and user service
	tokSvc := token.NewMemoryService()
	userService := userservice.New(userStorage, logger, password.NewBcryptHasher(), tokSvc)
	propertyTypeService := propertytypeservice.New(propertyTypeStorage, logger)
	propertyService := propertyservice.New(propertyStorage, propertyTypeStorage, logger, geocoder.NewNoop())
	favoriteService := favoriteservice.New(favoriteStorage, logger)
	imageService := imageservice.New(imageStorage, logger, "")

	// HTTP handlers and server
	router := chi.NewRouter()
	// register handlers under sensible prefixes
	httpHandler.NewUserHandler(userService).Register(router, "/users")
	httpHandler.NewTokenHandler(tokSvc).Register(router, "/tokens")
	httpHandler.NewFavoriteHandler(favoriteService).Register(router, "/favorites")
	httpHandler.NewPropertyHandler(propertyService).Register(router, "/properties")
	httpHandler.NewPropertyTypeHandler(propertyTypeService).Register(router, "/property_types")
	httpHandler.NewImageHandler(imageService).Register(router, "/images")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{Addr: ":" + port, Handler: router}

	// start server
	go func() {
		logger.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	// graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctxShut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger.Info("shutting down server")
	if err := srv.Shutdown(ctxShut); err != nil {
		logger.Error("shutdown error", "err", err)
		os.Exit(1)
	}

}
