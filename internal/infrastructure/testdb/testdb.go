package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/pressly/goose/v3"
)

// TestDB holds information about the running test database container.
type TestDB struct {
	DSN       string
	Host      string
	Port      string
	terminate func()
}

// StartContainer starts a Postgres+PostGIS container, runs migrations using goose library and returns TestDB and error.
// Caller must call the returned TestDB.Terminate() when done.
func StartContainer(ctx context.Context, logger *slog.Logger) (*TestDB, error) {
	logger.Info("starting test postgres container (postgis)")

	// If TEST_DSN is provided (CI) prefer using it instead of starting a docker container.
	if dsn := os.Getenv("TEST_DSN"); dsn != "" {
		logger.Info("TEST_DSN detected, using provided DSN instead of starting container", "dsn", dsn)
		// run migrations against provided DSN
		if err := runGooseMigrations(logger, dsn); err != nil {
			return nil, fmt.Errorf("failed to run migrations against TEST_DSN: %w", err)
		}

		// try to extract host and port for compatibility
		host := ""
		port := ""
		if u, err := url.Parse(dsn); err == nil {
			hostPort := u.Host
			if strings.Contains(hostPort, ":") {
				parts := strings.Split(hostPort, ":")
				host = parts[0]
				port = parts[1]
			} else {
				host = hostPort
			}
		}

		return &TestDB{DSN: dsn, Host: host, Port: port, terminate: func() {}}, nil
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	// Pull and run container
	opts := &dockertest.RunOptions{
		Repository: "postgis/postgis",
		Tag:        "15-3.3",
		Env: []string{
			"POSTGRES_USER=root",
			"POSTGRES_PASSWORD=root",
			"POSTGRES_DB=test",
		},
		ExposedPorts: []string{"5432/tcp"},
	}

	resource, err := pool.RunWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("could not start resource: %w", err)
	}

	// terminate function
	terminate := func() {
		_ = pool.Purge(resource)
	}

	// get host and port
	hostPort := resource.GetHostPort("5432/tcp") // returns "localhost:32768"
	var host string
	var port string
	// try simple scan
	_, err = fmt.Sscanf(hostPort, "%s:%s", &host, &port)
	if err != nil {
		// fallback: split
		for i := len(hostPort) - 1; i >= 0; i-- {
			if hostPort[i] == ':' {
				host = hostPort[:i]
				port = hostPort[i+1:]
				break
			}
		}
	}

	dsn := fmt.Sprintf("postgres://root:root@%s:%s/test?sslmode=disable", host, port)

	// wait until db is ready
	if err := pool.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, dsn)
		if err != nil {
			return err
		}
		_ = conn.Close(ctx)
		return nil
	}); err != nil {
		terminate()
		return nil, fmt.Errorf("could not connect to docker database: %w", err)
	}

	// Run migrations using goose (library) against the container DSN
	if err := runGooseMigrations(logger, dsn); err != nil {
		terminate()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &TestDB{
		DSN:       dsn,
		Host:      host,
		Port:      port,
		terminate: terminate,
	}, nil
}

func (t *TestDB) Terminate() {
	if t.terminate != nil {
		t.terminate()
	}
}

var (
	once     sync.Once
	instance *TestDB
	instErr  error
)

// EnsureStarted starts a singleton test container (only once) and returns the TestDB instance.
// It's safe to call from multiple packages; the container will be started only once.
func EnsureStarted(ctx context.Context, logger *slog.Logger) (*TestDB, error) {
	once.Do(func() {
		instance, instErr = StartContainer(ctx, logger)
	})
	return instance, instErr
}

func runGooseMigrations(logger *slog.Logger, dsn string) error {
	migrationsPath, err := getMigrationsPath(logger)
	if err != nil {
		return fmt.Errorf("failed to determine migrations path: %w", err)
	}

	logger.Info("running migrations (test container)", "path", migrationsPath, "dsn", dsn)

	// open database/sql DB using pgx stdlib driver
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open DB connection: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		// SetDialect returns error only on invalid dialect, but handle defensively
		logger.Error("failed to set goose dialect", "error", err)
	}

	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("failed to run migrations (goose): %w", err)
	}

	logger.Info("migrations applied successfully (test container)")
	return nil
}

func getMigrationsPath(logger *slog.Logger) (string, error) {
	possiblePaths := []string{
		"./db/migrations",
		"db/migrations",
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
