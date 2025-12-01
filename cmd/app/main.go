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
	"github.com/Oleja123/estate-agency/internal/handler/auth"

	propertyhandler "github.com/Oleja123/estate-agency/internal/handler/property"
	propertytypehandler "github.com/Oleja123/estate-agency/internal/handler/property_type"
	userhandler "github.com/Oleja123/estate-agency/internal/handler/user"
	postgresqlclient "github.com/Oleja123/estate-agency/internal/infrastructure/client/postgresql"
	"github.com/Oleja123/estate-agency/internal/infrastructure/config"
	favoritedb "github.com/Oleja123/estate-agency/internal/infrastructure/favorite"
	geocoder "github.com/Oleja123/estate-agency/internal/infrastructure/geocoder"
	imagedb "github.com/Oleja123/estate-agency/internal/infrastructure/image"
	propertydb "github.com/Oleja123/estate-agency/internal/infrastructure/property"
	propertytypedb "github.com/Oleja123/estate-agency/internal/infrastructure/property_type"
	userdb "github.com/Oleja123/estate-agency/internal/infrastructure/user"

	swagger "github.com/Oleja123/estate-agency/internal/handler/swagger"
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
	imageStorage := imagedb.New(dbClient, logger)
	propertyTypeStorage := propertytypedb.New(dbClient, logger)

	tokSvc := token.NewMemoryService()
	userService := userservice.New(userStorage, logger, password.NewBcryptHasher(), tokSvc)
	propertyTypeService := propertytypeservice.New(propertyTypeStorage, logger)
	favoriteService := favoriteservice.New(favoriteStorage, logger)
	// construct geocoder: use Geoapify if api key provided, otherwise noop
	var geoSvc geocoder.GeoService
	if cfg.GeoConfig.APIKey != "" {
		httpClient := &http.Client{Timeout: 10 * time.Second}
		geoSvc = geocoder.NewGeoapify(cfg.GeoConfig, httpClient)
	} else {
		geoSvc = geocoder.NewNoop()
	}

	userService.SetUserRole(context.Background(), 1, "admin")

	propertyService := propertyservice.New(propertyStorage, propertyTypeStorage, logger, geoSvc, favoriteService)
	imageService := imageservice.New(imageStorage, logger, "images/")

	router := chi.NewRouter()

	auth.SetLogger(logger)

	authMw := auth.AuthMiddleware(tokSvc)

	uh := userhandler.NewUserHandler(userService, logger, favoriteService)
	uh.Register(router, "/users", authMw)

	auth.NewTokenHandler(tokSvc, userService).Register(router, "/tokens", authMw)

	ph := propertyhandler.NewPropertyHandler(propertyService, logger, favoriteService, imageService)
	ph.Register(router, "/properties", authMw)

	propertytypehandler.NewPropertyTypeHandler(propertyTypeService, logger).Register(router, "/property_types", authMw)

	swagger.RegisterSwagger(router, logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{Addr: ":" + port, Handler: router}

	go func() {
		logger.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

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
