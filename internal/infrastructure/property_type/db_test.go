package propertytypedb

import (
	"context"
	"log/slog"
	"os"
	"testing"

	propertytype "github.com/Oleja123/estate-agency/internal/domain/property_type"
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

	// Ensure a single test DB is started (or use TEST_DSN if provided).
	tdb, err := testdb.EnsureStarted(testCtx, testLogger)
	if err != nil {
		testLogger.Error("Failed to start test DB container", "error", err)
		os.Exit(1)
	}
	defer tdb.Terminate()
	// update config to point to the container/DSN
	testConfig.DbConfig.Host = tdb.Host
	testConfig.DbConfig.Port = tdb.Port

	testClient, _ = postgresqlclient.NewClient(context.Background(), *testLogger, testConfig)
	testRepo = New(testClient, testLogger)

	code := m.Run()
	os.Exit(code)
}

func truncateTables() error {
	_, err := testClient.Exec(context.Background(), "TRUNCATE TABLE property_types RESTART IDENTITY CASCADE")
	return err
}

func TestPropertyTypeCRUD(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() propertytype.PropertyType
		action   func(t *testing.T, pt propertytype.PropertyType) (propertytype.PropertyType, error)
		validate func(t *testing.T, pt propertytype.PropertyType, err error)
	}{
		{
			name: "create_and_get_property_type",
			setup: func() propertytype.PropertyType {
				return propertytype.PropertyType{
					Name: "apartment",
				}
			},
			action: func(t *testing.T, pt propertytype.PropertyType) (propertytype.PropertyType, error) {
				Id, err := testRepo.Create(testCtx, pt)
				if err != nil {
					return propertytype.PropertyType{}, err
				}
				return testRepo.GetByID(testCtx, Id)
			},
			validate: func(t *testing.T, pt propertytype.PropertyType, err error) {
				require.NoError(t, err)
				assert.Equal(t, "apartment", pt.Name)
				assert.NotZero(t, pt.Id)
				assert.NotZero(t, pt.CreatedAt)
			},
		},
		{
			name: "update_property_type_name",
			setup: func() propertytype.PropertyType {
				pt := propertytype.PropertyType{
					Name: "house",
				}
				Id, err := testRepo.Create(testCtx, pt)
				require.NoError(t, err)

				created, err := testRepo.GetByID(testCtx, Id)
				require.NoError(t, err)
				return created
			},
			action: func(t *testing.T, pt propertytype.PropertyType) (propertytype.PropertyType, error) {
				pt.Name = "updated-house"
				err := testRepo.Update(testCtx, pt)
				if err != nil {
					return propertytype.PropertyType{}, err
				}
				return testRepo.GetByID(testCtx, pt.Id)
			},
			validate: func(t *testing.T, pt propertytype.PropertyType, err error) {
				require.NoError(t, err)
				assert.Equal(t, "updated-house", pt.Name)
			},
		},
		{
			name: "delete_property_type",
			setup: func() propertytype.PropertyType {
				pt := propertytype.PropertyType{
					Name: "to-delete",
				}
				Id, err := testRepo.Create(testCtx, pt)
				require.NoError(t, err)

				created, err := testRepo.GetByID(testCtx, Id)
				require.NoError(t, err)
				return created
			},
			action: func(t *testing.T, pt propertytype.PropertyType) (propertytype.PropertyType, error) {
				err := testRepo.Delete(testCtx, pt.Id)
				if err != nil {
					return propertytype.PropertyType{}, err
				}
				return testRepo.GetByID(testCtx, pt.Id)
			},
			validate: func(t *testing.T, pt propertytype.PropertyType, err error) {
				assert.Error(t, err)
				assert.True(t, basedb.IsNotFound(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())

			propertyType := tt.setup()
			result, err := tt.action(t, propertyType)
			tt.validate(t, result, err)
		})
	}
}

func TestPropertyTypeList(t *testing.T) {
	setupTestData := func() []int {
		types := []propertytype.PropertyType{
			{Name: "apartment"},
			{Name: "house"},
			{Name: "commercial"},
			{Name: "land"},
		}

		var ids []int
		for _, pt := range types {
			Id, err := testRepo.Create(testCtx, pt)
			require.NoError(t, err)
			ids = append(ids, Id)
		}
		return ids
	}

	tests := []struct {
		name     string
		request  propertytype.ListRequest
		wantLen  int
		validate func(t *testing.T, types []propertytype.PropertyType)
	}{
		{
			name: "get_all_types",
			request: propertytype.ListRequest{
				Limit: 10,
			},
			wantLen: 4,
		},
		{
			name: "search_by_name",
			request: propertytype.ListRequest{
				Filter: propertytype.Filter{
					Search: "apart",
				},
				Limit: 10,
			},
			wantLen: 1,
			validate: func(t *testing.T, types []propertytype.PropertyType) {
				assert.Equal(t, "apartment", types[0].Name)
			},
		},
		{
			name: "pagination_first_page",
			request: propertytype.ListRequest{
				Limit:  2,
				Offset: 0,
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())
			setupTestData()

			result, err := testRepo.List(testCtx, tt.request)
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
