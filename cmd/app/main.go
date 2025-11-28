// @title Estate Agency API
// @version 0.0.0
// @description API for Estate Agency service
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token as: "Bearer <token>" (include the word Bearer and a space)
package main

import (
	"context"
	"io"
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

	// imagehandler "github.com/Oleja123/estate-agency/internal/handler/image"
	propertyhandler "github.com/Oleja123/estate-agency/internal/handler/property"
	propertytypehandler "github.com/Oleja123/estate-agency/internal/handler/property_type"
	userhandler "github.com/Oleja123/estate-agency/internal/handler/user"
	postgresqlclient "github.com/Oleja123/estate-agency/internal/infrastructure/client/postgresql"
	"github.com/Oleja123/estate-agency/internal/infrastructure/config"
	favoritedb "github.com/Oleja123/estate-agency/internal/infrastructure/favorite"
	geocoder "github.com/Oleja123/estate-agency/internal/infrastructure/geocoder"
	propertydb "github.com/Oleja123/estate-agency/internal/infrastructure/property"
	propertytypedb "github.com/Oleja123/estate-agency/internal/infrastructure/property_type"
	userdb "github.com/Oleja123/estate-agency/internal/infrastructure/user"

	// swagger
	docs "github.com/Oleja123/estate-agency/docs"
	httpSwagger "github.com/swaggo/http-swagger"
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
	favoriteService := favoriteservice.New(favoriteStorage, logger)
	propertyService := propertyservice.New(propertyStorage, propertyTypeStorage, logger, geocoder.NewNoop(), favoriteService)
	imageService := imageservice.New(imageStorage, logger, "images/")

	// HTTP handlers and server
	router := chi.NewRouter()

	// set package logger for handler helpers/middlewares
	auth.SetLogger(logger)

	// create auth middleware (chi-style). We'll apply it per-route; public
	// endpoints (register/login/refresh) should be mounted without this
	// middleware.
	authMw := auth.AuthMiddleware(tokSvc)

	// register handlers under sensible prefixes; pass auth middleware so handlers
	// can apply it to protected routes using chi groups
	uh := userhandler.NewUserHandler(userService, logger, favoriteService)
	uh.Register(router, "/users", authMw)

	// Protect token endpoints by passing auth middleware. Per requirement,
	// all endpoints except /users/register and /users/login must require authorization.
	auth.NewTokenHandler(tokSvc).Register(router, "/tokens", authMw)

	// Register property handler and wire favorites toggle endpoint
	ph := propertyhandler.NewPropertyHandler(propertyService, logger, favoriteService, imageService)
	ph.Register(router, "/properties", authMw)

	// Do not register the standalone favorite handler: favorites endpoints are
	// now available under /users/{id}/favorites and /properties/{id}/favorites
	propertytypehandler.NewPropertyTypeHandler(propertyTypeService, logger).Register(router, "/property_types", authMw)
	// image handler removed; image endpoints are served under properties as needed

	// Serve swagger doc JSON and UI. The docs package is a stub initially; run
	// `swag init -g cmd/app/main.go -o docs` to generate full OpenAPI docs into
	// the `docs` package. The generated package will replace the stub.
	router.Get("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, docs.ReadDoc())
	})
	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

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
