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

type TestDB struct {
	DSN       string
	Host      string
	Port      string
	terminate func()
}

func StartContainer(ctx context.Context, logger *slog.Logger) (*TestDB, error) {
	logger.Info("starting test postgres container (postgis)")

	if dsn := os.Getenv("TEST_DSN"); dsn != "" {
		logger.Info("TEST_DSN detected, using provided DSN instead of starting container", "dsn", dsn)

		if err := runGooseMigrations(logger, dsn); err != nil {
			return nil, fmt.Errorf("failed to run migrations against TEST_DSN: %w", err)
		}

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

		if home, herr := os.UserHomeDir(); herr == nil {
			desktopSock := filepath.Join(home, ".docker", "desktop", "docker-cli.sock")
			if _, serr := os.Stat(desktopSock); serr == nil {

				_ = os.Setenv("DOCKER_HOST", "unix://"+desktopSock)
				pool, err = dockertest.NewPool("")
			}
		}
	}
	if err != nil {

		logger.Info("docker unavailable, trying local postgres at 127.0.0.1:5432")
		localDSN := "postgres://root:root@127.0.0.1:5432/test?sslmode=disable"

		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel2()
		conn, cerr := pgx.Connect(ctx2, localDSN)
		if cerr == nil {
			_ = conn.Close(ctx2)

			if err := runGooseMigrations(logger, localDSN); err != nil {
				return nil, fmt.Errorf("could not run migrations on local postgres: %w", err)
			}
			return &TestDB{DSN: localDSN, Host: "127.0.0.1", Port: "5432", terminate: func() {}}, nil
		}

		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

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

	terminate := func() {
		_ = pool.Purge(resource)
	}

	hostPort := resource.GetHostPort("5432/tcp")
	var host string
	var port string

	_, err = fmt.Sscanf(hostPort, "%s:%s", &host, &port)
	if err != nil {

		for i := len(hostPort) - 1; i >= 0; i-- {
			if hostPort[i] == ':' {
				host = hostPort[:i]
				port = hostPort[i+1:]
				break
			}
		}
	}

	dsn := fmt.Sprintf("postgres://root:root@%s:%s/test?sslmode=disable", host, port)

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

	logger.Info("migrations applied successfully (test container)")
	return nil
}

func getMigrationsPath(logger *slog.Logger) (string, error) {

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	maxDepth := 8
	dir := cwd
	for i := 0; i < maxDepth; i++ {
		candidate := filepath.Join(dir, "db", "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(candidate)
			logger.Info("migrations folder found", "path", absPath)
			return absPath, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	possiblePaths := []string{"./db/migrations", "db/migrations"}
	for _, p := range possiblePaths {
		absPath, _ := filepath.Abs(p)
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			logger.Info("migrations folder found (fallback)", "path", absPath)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("migrations folder not found: searched from %s and tried %v", cwd, possiblePaths)
}
