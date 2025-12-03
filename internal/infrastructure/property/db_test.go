package propertydb

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Oleja123/estate-agency/internal/domain/property"
	"github.com/Oleja123/estate-agency/internal/infrastructure/client"
	postgresqlclient "github.com/Oleja123/estate-agency/internal/infrastructure/client/postgresql"
	"github.com/Oleja123/estate-agency/internal/infrastructure/config"
	"github.com/Oleja123/estate-agency/internal/infrastructure/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testClient      client.Client
	testRepo        *Repository
	testLogger      *slog.Logger
	testCtx         context.Context
	testPropertyIDs []int
	testUserID      int
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

	testUserID = CreateTestUser()
	testPropertyIDs = CreateTestProperties()

	code := m.Run()
	os.Exit(code)
}

func CreateTestUser() int {

	_, _ = testClient.Exec(context.Background(), "TRUNCATE TABLE users RESTART IDENTITY CASCADE")

	var userID int
	err := testClient.QueryRow(context.Background(), `
        INSERT INTO users (email, password_hash, first_name, last_name, phone_number, user_role, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (email) DO NOTHING
        RETURNING id
    `, "test@example.com", "hash", "Test", "User", "+123456789", "client", true).Scan(&userID)

	if err != nil {

		if err.Error() == "sql: no rows in result set" {
			_ = testClient.QueryRow(context.Background(), "SELECT id FROM users WHERE email=$1", "test@example.com").Scan(&userID)
			return userID
		}
		panic("Failed to create test user: " + err.Error())
	}
	return userID
}

func CreateTestProperties() []int {

	_, _ = testClient.Exec(context.Background(), "TRUNCATE TABLE properties RESTART IDENTITY CASCADE")

	_, _ = testClient.Exec(context.Background(), `
        INSERT INTO property_types (property_name) VALUES
        ('apartment'), ('house'), ('commercial'), ('land')
        ON CONFLICT (property_name) DO NOTHING
    `)

	properties := []property.Property{
		{
			Title:               "Test Apartment 1",
			PropertyDescription: "Nice apartment",
			TypeID:              1,
			TransactionType:     property.TransactionSale,
			Price:               100000,
			Area:                75.0,
			PropertyAddress:     "Address 1",
			Latitude:            55.7558,
			Longitude:           37.6173,
			City:                "Moscow",
			PropertyStatus:      property.StatusActive,
			CreatedBy:           testUserID,
		},
		{
			Title:               "Test Apartment 2",
			PropertyDescription: "Another nice apartment",
			TypeID:              1,
			TransactionType:     property.TransactionRent,
			Price:               2000,
			Area:                60.0,
			PropertyAddress:     "Address 2",
			Latitude:            55.7558,
			Longitude:           37.6173,
			City:                "Moscow",
			PropertyStatus:      property.StatusActive,
			CreatedBy:           testUserID,
		},
	}

	var ids []int
	for _, prop := range properties {
		var id int
		err := testClient.QueryRow(context.Background(), `
            INSERT INTO properties (title, property_description, type_id, transaction_type, price, area, property_address, latitude, longitude, city, property_status, created_by)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            RETURNING id
        `, prop.Title, prop.PropertyDescription, prop.TypeID, prop.TransactionType, prop.Price, prop.Area, prop.PropertyAddress, prop.Latitude, prop.Longitude, prop.City, prop.PropertyStatus, prop.CreatedBy).Scan(&id)

		if err != nil {
			panic("Failed to create test property: " + err.Error())
		}
		ids = append(ids, id)
	}
	return ids
}

func TruncateTables() error {
	_, err := testClient.Exec(context.Background(), "TRUNCATE TABLE favorites RESTART IDENTITY CASCADE")
	return err
}

func TestGetByIDWithFavorite(t *testing.T) {
	// ensure clean favorites
	require.NoError(t, TruncateTables())

	// without favorite
	p, fav, err := testRepo.GetByIDWithFavorite(testCtx, testPropertyIDs[0], testUserID)
	require.NoError(t, err)
	assert.Equal(t, testPropertyIDs[0], p.ID)
	assert.False(t, fav)

	// add favorite row
	_, err = testClient.Exec(context.Background(), `INSERT INTO favorites (user_id, property_id) VALUES ($1,$2)`, testUserID, testPropertyIDs[0])
	require.NoError(t, err)

	p2, fav2, err := testRepo.GetByIDWithFavorite(testCtx, testPropertyIDs[0], testUserID)
	require.NoError(t, err)
	assert.Equal(t, testPropertyIDs[0], p2.ID)
	assert.True(t, fav2)
}
