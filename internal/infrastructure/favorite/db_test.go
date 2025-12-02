package favoritedb

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Oleja123/estate-agency/internal/domain/favorite"
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
	testClient      client.Client
	testRepo        *Repository
	testLogger      *slog.Logger
	testCtx         context.Context
	testUserID      int
	testPropertyIDs []int
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
			City:                "Moscow",
			PropertyStatus:      property.StatusActive,
			CreatedBy:           testUserID,
		},
	}

	var ids []int
	for _, prop := range properties {
		var id int
		err := testClient.QueryRow(context.Background(), `
            INSERT INTO properties (title, property_description, type_id, transaction_type, price, area, property_address, city, property_status, created_by)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING id
        `, prop.Title, prop.PropertyDescription, prop.TypeID, prop.TransactionType, prop.Price, prop.Area, prop.PropertyAddress, prop.City, prop.PropertyStatus, prop.CreatedBy).Scan(&id)

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

func TestFavoriteCRUD(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() favorite.Favorite
		action   func(t *testing.T, fav favorite.Favorite) (property.Property, error)
		validate func(t *testing.T, fav property.Property, err error)
	}{
		{
			name: "create_and_get_favorite",
			setup: func() favorite.Favorite {
				return favorite.Favorite{
					UserID:     testUserID,
					PropertyID: testPropertyIDs[0],
				}
			},
			action: func(t *testing.T, fav favorite.Favorite) (property.Property, error) {
				err := testRepo.Create(testCtx, fav)
				if err != nil {
					return property.Property{}, err
				}
				return testRepo.GetByUserAndProperty(testCtx, fav.UserID, fav.PropertyID)
			},
			validate: func(t *testing.T, fav property.Property, err error) {
				require.NoError(t, err)
				assert.Equal(t, testUserID, fav.CreatedBy)
				assert.Equal(t, testPropertyIDs[0], fav.ID)
				assert.NotZero(t, fav.CreatedAt)
			},
		},
		{
			name: "delete_favorite",
			setup: func() favorite.Favorite {
				fav := favorite.Favorite{
					UserID:     testUserID,
					PropertyID: testPropertyIDs[0],
				}
				err := testRepo.Create(testCtx, fav)
				require.NoError(t, err)
				return fav
			},
			action: func(t *testing.T, fav favorite.Favorite) (property.Property, error) {
				_, err := testRepo.Delete(testCtx, fav.UserID, fav.PropertyID)
				if err != nil {
					return property.Property{}, err
				}
				return testRepo.GetByUserAndProperty(testCtx, fav.UserID, fav.PropertyID)
			},
			validate: func(t *testing.T, fav property.Property, err error) {
				assert.Error(t, err)
				assert.True(t, basedb.IsNotFound(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, TruncateTables())

			favorite := tt.setup()
			result, err := tt.action(t, favorite)
			tt.validate(t, result, err)
		})
	}
}

func TestFavoriteList(t *testing.T) {
	setupTestData := func() {
		favorites := []favorite.Favorite{
			{UserID: testUserID, PropertyID: testPropertyIDs[0]},
			{UserID: testUserID, PropertyID: testPropertyIDs[1]},
		}

		for _, fav := range favorites {
			err := testRepo.Create(testCtx, fav)
			require.NoError(t, err)
		}
	}

	tests := []struct {
		name      string
		request   favorite.ListRequest
		wantLen   int
		wantTotal int
		validate  func(t *testing.T, favorites []property.Property)
	}{
		{
			name: "get_all_favorites",
			request: favorite.ListRequest{
				Limit: 10,
			},
			wantLen:   2,
			wantTotal: 2,
		},
		{
			name: "filter_by_user",
			request: favorite.ListRequest{
				Filter: favorite.Filter{UserID: testUserID},
				Limit:  10,
			},
			wantLen:   2,
			wantTotal: 2,
			validate: func(t *testing.T, favorites []property.Property) {
				for _, fav := range favorites {
					assert.Equal(t, testUserID, fav.CreatedBy)
				}
			},
		},
		{
			name: "filter_by_property",
			request: favorite.ListRequest{
				Filter: favorite.Filter{PropertyID: testPropertyIDs[0]},
				Limit:  10,
			},
			wantLen:   1,
			wantTotal: 1,
			validate: func(t *testing.T, favorites []property.Property) {
				assert.Equal(t, testPropertyIDs[0], favorites[0].ID)
			},
		},
		{
			name: "pagination",
			request: favorite.ListRequest{
				Limit:  1,
				Offset: 0,
			},
			wantLen:   1,
			wantTotal: 2,
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

func TestFavoriteExists(t *testing.T) {
	t.Run("check_favorite_exists", func(t *testing.T) {
		require.NoError(t, TruncateTables())

		fav := favorite.Favorite{
			UserID:     testUserID,
			PropertyID: testPropertyIDs[0],
		}

		exists, err := testRepo.Exists(testCtx, fav.UserID, fav.PropertyID)
		require.NoError(t, err)
		assert.False(t, exists)

		err = testRepo.Create(testCtx, fav)
		require.NoError(t, err)

		exists, err = testRepo.Exists(testCtx, fav.UserID, fav.PropertyID)
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestGetByUser(t *testing.T) {
	t.Run("get_favorites_by_user_via_list", func(t *testing.T) {
		require.NoError(t, TruncateTables())

		favorites := []favorite.Favorite{
			{UserID: testUserID, PropertyID: testPropertyIDs[0]},
			{UserID: testUserID, PropertyID: testPropertyIDs[1]},
		}

		for _, fav := range favorites {
			err := testRepo.Create(testCtx, fav)
			require.NoError(t, err)
		}

		result, total, err := testRepo.List(testCtx, favorite.ListRequest{
			Filter: favorite.Filter{UserID: testUserID},
			Limit:  100,
		})
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, 2, total)

		for _, fav := range result {
			assert.Equal(t, testUserID, fav.CreatedBy)
		}
	})
}

func TestGetByProperty(t *testing.T) {
	t.Run("get_favorites_by_property_via_list", func(t *testing.T) {
		require.NoError(t, TruncateTables())

		fav := favorite.Favorite{
			UserID:     testUserID,
			PropertyID: testPropertyIDs[0],
		}

		err := testRepo.Create(testCtx, fav)
		require.NoError(t, err)

		result, total, err := testRepo.List(testCtx, favorite.ListRequest{
			Filter: favorite.Filter{PropertyID: testPropertyIDs[0]},
			Limit:  100,
		})
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, testPropertyIDs[0], result[0].ID)
		assert.Equal(t, 1, total)
	})
}
