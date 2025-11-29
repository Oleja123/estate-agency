package imagedb

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Oleja123/estate-agency/internal/domain/image"
	"github.com/Oleja123/estate-agency/internal/infrastructure/basedb"
	"github.com/Oleja123/estate-agency/internal/infrastructure/client"
	postgresqlclient "github.com/Oleja123/estate-agency/internal/infrastructure/client/postgresql"
	"github.com/Oleja123/estate-agency/internal/infrastructure/config"
	"github.com/Oleja123/estate-agency/internal/infrastructure/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testClient client.Client
	testRepo   *ImageRepository
	testLogger *slog.Logger
	testCtx    context.Context
	testUserID int
)

func TestMain(m *testing.M) {
	testCtx = context.Background()

	testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	testConfig := config.Config{
		DbConfig: config.DatabaseConfig{
			Username:         "root",
			Password:         "root",
			Host:             "localhost",
			Port:             "5432",
			Database:         "test",
			MaxAttempts:      5,
			SecondsToConnect: 5,
		},
		GoosePath: "/home/oleg/go/bin/goose",
	}

	tdb, err := testdb.EnsureStarted(testCtx, testLogger)
	if err != nil {
		testLogger.Error("Failed to start test DB container", "error", err)
		os.Exit(1)
	}
	defer tdb.Terminate()
	testConfig.DbConfig.Host = tdb.Host
	testConfig.DbConfig.Port = tdb.Port

	testClient, _ = postgresqlclient.NewClient(context.Background(), *testLogger, testConfig)
	testRepo = New(testClient, testLogger)

	testUserID = createTestUserForImageTests()

	code := m.Run()
	os.Exit(code)
}

func createTestUserForImageTests() int {
	_, _ = testClient.Exec(context.Background(), "TRUNCATE TABLE users RESTART IDENTITY CASCADE")

	var userID int
	err := testClient.QueryRow(context.Background(), `
        INSERT INTO users (email, password_hash, first_name, last_name, phone_number, user_role, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (email) DO NOTHING
        RETURNING id
    `, "test-image@example.com", "hash", "Test", "User", "+123456789", "client", true).Scan(&userID)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			_ = testClient.QueryRow(context.Background(), "SELECT id FROM users WHERE email=$1", "test-image@example.com").Scan(&userID)
			return userID
		}
		panic("Failed to create test user: " + err.Error())
	}
	return userID
}

func createTestProperty(t *testing.T) int {
	t.Helper()
	_, _ = testClient.Exec(context.Background(), "TRUNCATE TABLE properties RESTART IDENTITY CASCADE")

	var id int
	err := testClient.QueryRow(context.Background(), `
        INSERT INTO properties (title, property_description, type_id, transaction_type, price, area, property_address, latitude, longitude, city, property_status, created_by, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())
        RETURNING id
    `, "Img Test", "desc", 1, "sale", 1000, 50.0, "addr", 0.0, 0.0, "City", "active", testUserID).Scan(&id)
	require.NoError(t, err)
	return id
}

func truncateImages() error {
	_, err := testClient.Exec(context.Background(), "TRUNCATE TABLE property_images RESTART IDENTITY CASCADE")
	return err
}

func TestImageCRUD(t *testing.T) {
	t.Run("create_get_list_delete", func(t *testing.T) {
		require.NoError(t, truncateImages())

		propID := createTestProperty(t)

		img := image.PropertyImage{
			PropertyID: propID,
			Path:       "/tmp/image1.jpg",
		}

		imageRepo := testRepo

		id, err := imageRepo.Create(testCtx, img)
		require.NoError(t, err)
		require.Greater(t, id, 0)

		got, err := imageRepo.GetByID(testCtx, id)
		require.NoError(t, err)
		assert.Equal(t, propID, got.PropertyID)
		assert.Equal(t, img.Path, got.Path)

		list, err := imageRepo.ListByProperty(testCtx, propID)
		require.NoError(t, err)
		require.Len(t, list, 1)

		_, err = imageRepo.Delete(testCtx, id)
		require.NoError(t, err)

		_, err = imageRepo.GetByID(testCtx, id)
		assert.Error(t, err)
		assert.True(t, basedb.IsNotFound(err))
	})

	t.Run("create_many", func(t *testing.T) {
		require.NoError(t, truncateImages())

		propID := createTestProperty(t)

		imgs := []image.PropertyImage{
			{PropertyID: propID, Path: "/tmp/image_a.jpg"},
			{PropertyID: propID, Path: "/tmp/image_b.jpg"},
			{PropertyID: propID, Path: "/tmp/image_c.jpg"},
		}

		imageRepo := testRepo

		ids, err := imageRepo.CreateMany(testCtx, imgs)
		require.NoError(t, err)
		require.Len(t, ids, len(imgs))

		list, err := imageRepo.ListByProperty(testCtx, propID)
		require.NoError(t, err)
		require.Len(t, list, len(imgs))

		for _, id := range ids {
			_, err := imageRepo.Delete(testCtx, id)
			require.NoError(t, err)
		}
	})
}

func TestDeleteManyByProperty(t *testing.T) {
	require.NoError(t, truncateImages())

	propID := createTestProperty(t)

	imgs := []image.PropertyImage{
		{PropertyID: propID, Path: "/tmp/image_a.jpg"},
		{PropertyID: propID, Path: "/tmp/image_b.jpg"},
	}

	imageRepo := testRepo

	ids, err := imageRepo.CreateMany(testCtx, imgs)
	require.NoError(t, err)
	require.Len(t, ids, len(imgs))

	deletedIDs, err := imageRepo.DeleteMany(testCtx, propID)
	require.NoError(t, err)
	require.Len(t, deletedIDs, len(imgs))

	list, err := imageRepo.ListByProperty(testCtx, propID)
	require.NoError(t, err)
	require.Len(t, list, 0)

	for _, id := range ids {
		_, err := imageRepo.GetByID(testCtx, id)
		require.Error(t, err)
	}
}

func TestDeleteMany_InvalidProperty(t *testing.T) {
	require.NoError(t, truncateImages())

	imageRepo := testRepo

	_, err := imageRepo.DeleteMany(testCtx, 99999)
	require.Error(t, err)

	assert.True(t, basedb.IsNotFound(err))
}
