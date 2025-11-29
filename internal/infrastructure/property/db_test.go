package propertydb

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Oleja123/estate-agency/internal/domain/property"
	"github.com/Oleja123/estate-agency/internal/domain/user"
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
	testRepo   *Repository
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

	testUserID = CreateTestUser()

	CreateDefaultPropertyTypes()

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
    `, "test@example.com", "hash", "Test", "User", "+123456789", user.RoleClient, true).Scan(&userID)

	if err != nil {

		if err.Error() == "sql: no rows in result set" {
			_ = testClient.QueryRow(context.Background(), "SELECT id FROM users WHERE email=$1", "test@example.com").Scan(&userID)
			return userID
		}
		panic("Failed to create test user: " + err.Error())
	}
	return userID
}

func TruncateTables() error {
	_, err := testClient.Exec(context.Background(), "TRUNCATE TABLE properties RESTART IDENTITY CASCADE")
	return err
}

func CreateDefaultPropertyTypes() {

	_, _ = testClient.Exec(context.Background(), "TRUNCATE TABLE property_types RESTART IDENTITY CASCADE")

	types := []string{"apartment", "house", "commercial", "land"}
	for _, name := range types {
		_, err := testClient.Exec(context.Background(), `
            INSERT INTO property_types (property_name) VALUES ($1)
            ON CONFLICT (property_name) DO NOTHING
        `, name)
		if err != nil {
			panic("Failed to create test property_types: " + err.Error())
		}
	}
}

func TestPropertyCRUD(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() property.Property
		action   func(t *testing.T, prop property.Property) (property.Property, error)
		validate func(t *testing.T, prop property.Property, err error)
	}{
		{
			name: "create_and_get_property",
			setup: func() property.Property {
				return property.Property{
					Title:               "Test Apartment",
					PropertyDescription: "Nice apartment in city center",
					TypeID:              1,
					TransactionType:     property.TransactionSale,
					Price:               100000.50,
					Area:                75.5,
					PropertyAddress:     "123 Main St, Moscow",
					Latitude:            55.7558,
					Longitude:           37.6173,
					City:                "Moscow",
					PropertyStatus:      property.StatusActive,
					CreatedBy:           testUserID,
				}
			},
			action: func(t *testing.T, prop property.Property) (property.Property, error) {
				id, err := testRepo.Create(testCtx, prop)
				if err != nil {
					return property.Property{}, err
				}
				return testRepo.GetByID(testCtx, id)
			},
			validate: func(t *testing.T, prop property.Property, err error) {
				require.NoError(t, err)
				assert.Equal(t, "Test Apartment", prop.Title)
				assert.Equal(t, property.TransactionSale, prop.TransactionType)
				assert.Equal(t, 100000.50, prop.Price)
				assert.Equal(t, 75.5, prop.Area)
				assert.Equal(t, "Moscow", prop.City)
				assert.Equal(t, property.StatusActive, prop.PropertyStatus)
				assert.NotZero(t, prop.ID)
				assert.NotZero(t, prop.CreatedAt)
			},
		},
		{
			name: "update_property",
			setup: func() property.Property {
				prop := property.Property{
					Title:               "Old Title",
					PropertyDescription: "Old description",
					TypeID:              1,
					TransactionType:     property.TransactionSale,
					Price:               50000,
					Area:                50.0,
					PropertyAddress:     "Old address",
					City:                "Old City",
					PropertyStatus:      property.StatusActive,
					CreatedBy:           testUserID,
				}
				id, err := testRepo.Create(testCtx, prop)
				require.NoError(t, err)

				created, err := testRepo.GetByID(testCtx, id)
				require.NoError(t, err)
				return created
			},
			action: func(t *testing.T, prop property.Property) (property.Property, error) {
				prop.Title = "Updated Title"
				prop.PropertyDescription = "Updated description"
				prop.Price = 75000
				prop.PropertyStatus = property.StatusSold
				err := testRepo.Update(testCtx, prop)
				if err != nil {
					return property.Property{}, err
				}
				return testRepo.GetByID(testCtx, prop.ID)
			},
			validate: func(t *testing.T, prop property.Property, err error) {
				require.NoError(t, err)
				assert.Equal(t, "Updated Title", prop.Title)
				assert.Equal(t, "Updated description", prop.PropertyDescription)
				assert.Equal(t, 75000.0, prop.Price)
				assert.Equal(t, property.StatusSold, prop.PropertyStatus)
			},
		},
		{
			name: "delete_property",
			setup: func() property.Property {
				prop := property.Property{
					Title:               "To Delete",
					PropertyDescription: "Will be deleted",
					TypeID:              1,
					TransactionType:     property.TransactionRent,
					Price:               1000,
					Area:                30.0,
					PropertyAddress:     "Delete address",
					City:                "Delete City",
					PropertyStatus:      property.StatusActive,
					CreatedBy:           testUserID,
				}
				id, err := testRepo.Create(testCtx, prop)
				require.NoError(t, err)

				created, err := testRepo.GetByID(testCtx, id)
				require.NoError(t, err)
				return created
			},
			action: func(t *testing.T, prop property.Property) (property.Property, error) {
				_, err := testRepo.Delete(testCtx, prop.ID)
				if err != nil {
					return property.Property{}, err
				}
				return testRepo.GetByID(testCtx, prop.ID)
			},
			validate: func(t *testing.T, prop property.Property, err error) {
				assert.Error(t, err)
				assert.True(t, basedb.IsNotFound(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, TruncateTables())

			property := tt.setup()
			result, err := tt.action(t, property)
			tt.validate(t, result, err)
		})
	}
}

func TestPropertyList(t *testing.T) {
	setupTestData := func() []int {
		properties := []property.Property{
			{
				Title:               "Apartment for sale",
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
				Title:               "House for rent",
				PropertyDescription: "Big house",
				TypeID:              2,
				TransactionType:     property.TransactionRent,
				Price:               2000,
				Area:                150.0,
				PropertyAddress:     "Address 2",
				Latitude:            55.7600,
				Longitude:           37.6200,
				City:                "Moscow",
				PropertyStatus:      property.StatusActive,
				CreatedBy:           testUserID,
			},
			{
				Title:               "Commercial space",
				PropertyDescription: "Office space",
				TypeID:              3,
				TransactionType:     property.TransactionSale,
				Price:               500000,
				Area:                200.0,
				PropertyAddress:     "Address 3",
				Latitude:            55.7500,
				Longitude:           37.6000,
				City:                "St Petersburg",
				PropertyStatus:      property.StatusSold,
				CreatedBy:           testUserID,
			},
		}

		var ids []int
		for _, prop := range properties {
			id, err := testRepo.Create(testCtx, prop)
			require.NoError(t, err)
			ids = append(ids, id)
		}
		return ids
	}

	tests := []struct {
		name      string
		request   property.ListRequest
		wantLen   int
		wantTotal int
		validate  func(t *testing.T, properties []property.Property)
	}{
		{
			name: "get_all_properties",
			request: property.ListRequest{
				Limit: 10,
			},
			wantLen:   3,
			wantTotal: 3,
		},
		{
			name: "filter_by_transaction_type",
			request: property.ListRequest{
				Filter: property.Filter{
					TransactionType: property.TransactionSale,
				},
				Limit: 10,
			},
			wantLen:   2,
			wantTotal: 2,
			validate: func(t *testing.T, properties []property.Property) {
				for _, prop := range properties {
					assert.Equal(t, property.TransactionSale, prop.TransactionType)
				}
			},
		},
		{
			name: "filter_by_city",
			request: property.ListRequest{
				Filter: property.Filter{
					City: "Moscow",
				},
				Limit: 10,
			},
			wantLen:   2,
			wantTotal: 2,
			validate: func(t *testing.T, properties []property.Property) {
				for _, prop := range properties {
					assert.Equal(t, "Moscow", prop.City)
				}
			},
		},
		{
			name: "filter_by_status",
			request: property.ListRequest{
				Filter: property.Filter{
					PropertyStatus: property.StatusActive,
				},
				Limit: 10,
			},
			wantLen:   2,
			wantTotal: 2,
			validate: func(t *testing.T, properties []property.Property) {
				for _, prop := range properties {
					assert.Equal(t, property.StatusActive, prop.PropertyStatus)
				}
			},
		},
		{
			name: "filter_by_price",
			request: property.ListRequest{
				Filter: property.Filter{
					MinPrice: 1000,
					MaxPrice: 100000,
				},
				Limit: 10,
			},
			wantLen:   2,
			wantTotal: 2,
			validate: func(t *testing.T, properties []property.Property) {
				for _, prop := range properties {
					assert.True(t, prop.Price >= 1000 && prop.Price <= 100000)
				}
			},
		},
		{
			name: "search_by_text",
			request: property.ListRequest{
				Filter: property.Filter{
					Search: "apartment",
				},
				Limit: 10,
			},
			wantLen:   1,
			wantTotal: 1,
			validate: func(t *testing.T, properties []property.Property) {
				assert.Equal(t, "Apartment for sale", properties[0].Title)
			},
		},
		{
			name: "pagination_first_page",
			request: property.ListRequest{
				Limit:  2,
				Offset: 0,
			},
			wantLen:   2,
			wantTotal: 3,
		},
		{
			name: "pagination_second_page",
			request: property.ListRequest{
				Limit:  2,
				Offset: 2,
			},
			wantLen:   1,
			wantTotal: 3,
		},
		{
			name: "filter_by_creator",
			request: property.ListRequest{
				Filter: property.Filter{
					CreatedBy: testUserID,
				},
				Limit: 10,
			},
			wantLen:   3,
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, TruncateTables())
			setupTestData()

			result, total, err := testRepo.List(testCtx, tt.request)
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)
			if tt.wantTotal != 0 {
				assert.Equal(t, tt.wantTotal, total)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestPropertyListWithDistanceFilter(t *testing.T) {
	setupTestData := func() {
		properties := []property.Property{
			{
				Title:               "Close property",
				PropertyDescription: "Very close to center",
				TypeID:              1,
				TransactionType:     property.TransactionSale,
				Price:               100000,
				Area:                75.0,
				PropertyAddress:     "Near center",
				Latitude:            55.7558,
				Longitude:           37.6173,
				City:                "Moscow",
				PropertyStatus:      property.StatusActive,
				CreatedBy:           testUserID,
			},
			{
				Title:               "Far property",
				PropertyDescription: "Far from center",
				TypeID:              1,
				TransactionType:     property.TransactionSale,
				Price:               80000,
				Area:                80.0,
				PropertyAddress:     "Far away",
				Latitude:            55.8000,
				Longitude:           37.8000,
				City:                "Moscow",
				PropertyStatus:      property.StatusActive,
				CreatedBy:           testUserID,
			},
		}

		for _, prop := range properties {
			_, err := testRepo.Create(testCtx, prop)
			require.NoError(t, err)
		}
	}

	t.Run("filter_by_distance", func(t *testing.T) {
	require.NoError(t, TruncateTables())
		setupTestData()

		request := property.ListRequest{
			Filter: property.Filter{
				Latitude:  55.7558,
				Longitude: 37.6173,
				RadiusKm:  10.0,
			},
			Limit: 10,
		}

		result, total, err := testRepo.List(testCtx, request)
		require.NoError(t, err)

		require.Len(t, result, 1)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Close property", result[0].Title)
	})
}
