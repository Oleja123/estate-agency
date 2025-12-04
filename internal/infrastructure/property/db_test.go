package propertydb

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/Oleja123/estate-agency/internal/domain/property"
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
		testLogger.Error("не удалось запустить тестовый контейнер базы данных", "error", err)
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
		panic("не удалось создать тестового пользователя: " + err.Error())
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
			panic("не удалось создать тестовое свойство: " + err.Error())
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

	require.NoError(t, TruncateTables())

	p, fav, err := testRepo.GetByIDWithFavorite(testCtx, testPropertyIDs[0], testUserID)
	require.NoError(t, err)
	assert.Equal(t, testPropertyIDs[0], p.ID)
	assert.False(t, fav)

	_, err = testClient.Exec(context.Background(), `INSERT INTO favorites (user_id, property_id) VALUES ($1,$2)`, testUserID, testPropertyIDs[0])
	require.NoError(t, err)

	p2, fav2, err := testRepo.GetByIDWithFavorite(testCtx, testPropertyIDs[0], testUserID)
	require.NoError(t, err)
	assert.Equal(t, testPropertyIDs[0], p2.ID)
	assert.True(t, fav2)
}

func TestListLoadsThumbnail(t *testing.T) {

	_, _ = testClient.Exec(context.Background(), "TRUNCATE TABLE property_images RESTART IDENTITY CASCADE")

	var img1id, img2id, img3id int
	err := testClient.QueryRow(context.Background(), `INSERT INTO property_images (property_id, path, created_at) VALUES ($1, $2, NOW()) RETURNING id`, testPropertyIDs[0], "p1_img_a.jpg").Scan(&img1id)
	require.NoError(t, err)
	err = testClient.QueryRow(context.Background(), `INSERT INTO property_images (property_id, path, created_at) VALUES ($1, $2, NOW()) RETURNING id`, testPropertyIDs[0], "p1_img_b.jpg").Scan(&img2id)
	require.NoError(t, err)

	err = testClient.QueryRow(context.Background(), `INSERT INTO property_images (property_id, path, created_at) VALUES ($1, $2, NOW()) RETURNING id`, testPropertyIDs[1], "p2_img_a.jpg").Scan(&img3id)
	require.NoError(t, err)

	props, total, err := testRepo.List(testCtx, property.ListRequest{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, len(props))

	var found1, found2 *property.Property
	for i := range props {
		if props[i].ID == testPropertyIDs[0] {
			found1 = &props[i]
		}
		if props[i].ID == testPropertyIDs[1] {
			found2 = &props[i]
		}
	}
	require.NotNil(t, found1)
	require.NotNil(t, found2)

	require.Len(t, found1.Images, 1)
	assert.Equal(t, img1id, found1.Images[0].ID)

	require.Len(t, found2.Images, 1)
	assert.Equal(t, img3id, found2.Images[0].ID)
}

func TestGetByID_NotFound(t *testing.T) {
	_, err := testRepo.GetByID(testCtx, 999999)
	assert.Error(t, err)
	assert.True(t, basedb.IsNotFound(err))
}

func TestUpdateDeleteFlow(t *testing.T) {

	p := property.Property{
		Title:               "Flow Property",
		PropertyDescription: "To be updated",
		TypeID:              1,
		TransactionType:     property.TransactionSale,
		Price:               50000,
		Area:                40,
		PropertyAddress:     "Flow Address",
		Latitude:            55.0,
		Longitude:           37.0,
		City:                "FlowCity",
		PropertyStatus:      property.StatusActive,
		CreatedBy:           testUserID,
	}
	id, err := testRepo.Create(testCtx, p)
	require.NoError(t, err)

	p.ID = id
	p.Title = "Flow Property Updated"
	p.Price = 60000
	upd, err := testRepo.Update(testCtx, p)
	require.NoError(t, err)
	assert.Equal(t, "Flow Property Updated", upd.Title)
	assert.Equal(t, 60000.0, upd.Price)

	_, err = testRepo.Delete(testCtx, id)
	require.NoError(t, err)

	_, err = testRepo.GetByID(testCtx, id)
	assert.Error(t, err)
	assert.True(t, basedb.IsNotFound(err))
}

func TestListFiltering(t *testing.T) {

	p1 := property.Property{Title: "Filter A", PropertyDescription: "Desc A", TypeID: 2, TransactionType: property.TransactionSale, Price: 11111, Area: 10, PropertyAddress: "Addr A", Latitude: 10, Longitude: 10, City: "CityA", PropertyStatus: property.StatusActive, CreatedBy: testUserID}
	p2 := property.Property{Title: "Filter B", PropertyDescription: "Desc B", TypeID: 3, TransactionType: property.TransactionRent, Price: 22222, Area: 20, PropertyAddress: "Addr B", Latitude: 11, Longitude: 11, City: "CityB", PropertyStatus: property.StatusActive, CreatedBy: testUserID}
	p3 := property.Property{Title: "SearchMe", PropertyDescription: "specialsearch", TypeID: 2, TransactionType: property.TransactionSale, Price: 33333, Area: 30, PropertyAddress: "Addr C", Latitude: 12, Longitude: 12, City: "CityA", PropertyStatus: property.StatusActive, CreatedBy: testUserID}

	id1, err := testRepo.Create(testCtx, p1)
	require.NoError(t, err)
	id2, err := testRepo.Create(testCtx, p2)
	require.NoError(t, err)
	id3, err := testRepo.Create(testCtx, p3)
	require.NoError(t, err)

	tests := []struct {
		name    string
		filter  property.Filter
		wantIDs []int
	}{
		{"by_type", property.Filter{TypeIDs: []int{2}}, []int{id1, id3}},
		{"by_city", property.Filter{City: "CityB"}, []int{id2}},
		{"by_transaction", property.Filter{TransactionType: property.TransactionRent}, []int{id2}},
		{"by_price_range", property.Filter{MinPrice: 20000, MaxPrice: 40000}, []int{id2, id3}},
		{"by_area_range", property.Filter{MinArea: 15, MaxArea: 35}, []int{id2, id3}},
		{"by_search", property.Filter{Search: "SearchMe"}, []int{id3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props, _, err := testRepo.List(testCtx, property.ListRequest{Filter: tt.filter, Limit: 50, Offset: 0})
			require.NoError(t, err)
			gotIDs := make(map[int]bool)
			for _, p := range props {
				gotIDs[p.ID] = true
			}
			for _, want := range tt.wantIDs {
				assert.True(t, gotIDs[want], "expected id %d in results", want)
			}
		})
	}
}

func TestListGeoFilter_Radius(t *testing.T) {

	pCenter := property.Property{Title: "GeoCenter", PropertyDescription: "Near center", TypeID: 1, TransactionType: property.TransactionSale, Price: 100, Area: 10, PropertyAddress: "Origin", Latitude: 10.0, Longitude: 10.0, City: "GEO", PropertyStatus: property.StatusActive, CreatedBy: testUserID}

	pFar := property.Property{Title: "GeoFar", PropertyDescription: "Far away", TypeID: 1, TransactionType: property.TransactionSale, Price: 200, Area: 20, PropertyAddress: "Far", Latitude: 11.0, Longitude: 10.0, City: "GEO", PropertyStatus: property.StatusActive, CreatedBy: testUserID}

	idCenter, err := testRepo.Create(testCtx, pCenter)
	require.NoError(t, err)
	idFar, err := testRepo.Create(testCtx, pFar)
	require.NoError(t, err)

	props, _, err := testRepo.List(testCtx, property.ListRequest{Filter: property.Filter{Latitude: 10.0, Longitude: 10.0, RadiusKm: 20}, Limit: 50, Offset: 0})
	require.NoError(t, err)
	ids := make(map[int]bool)
	for _, p := range props {
		ids[p.ID] = true
	}
	assert.True(t, ids[idCenter])
	assert.False(t, ids[idFar])

	props2, _, err := testRepo.List(testCtx, property.ListRequest{Filter: property.Filter{Latitude: 10.0, Longitude: 10.0, RadiusKm: 150}, Limit: 50, Offset: 0})
	require.NoError(t, err)
	ids2 := make(map[int]bool)
	for _, p := range props2 {
		ids2[p.ID] = true
	}
	assert.True(t, ids2[idCenter])
	assert.True(t, ids2[idFar])
}

func TestListZeroLimit(t *testing.T) {

	props, total, err := testRepo.List(testCtx, property.ListRequest{Limit: 0, Offset: 0})
	require.NoError(t, err)

	assert.Equal(t, 0, len(props))
	assert.GreaterOrEqual(t, total, 0)
}

func TestDeleteNotFound(t *testing.T) {
	_, err := testRepo.Delete(testCtx, 9999999)
	assert.Error(t, err)
	assert.True(t, basedb.IsNotFound(err))
}

func TestListGeoFilter_ZeroCoordsAndNegativeRadius(t *testing.T) {

	pA := property.Property{Title: "GeoA", PropertyDescription: "A", TypeID: 1, TransactionType: property.TransactionSale, Price: 100, Area: 10, PropertyAddress: "A", Latitude: 20.0, Longitude: 20.0, City: "ZGeo", PropertyStatus: property.StatusActive, CreatedBy: testUserID}
	pB := property.Property{Title: "GeoB", PropertyDescription: "B", TypeID: 1, TransactionType: property.TransactionSale, Price: 200, Area: 20, PropertyAddress: "B", Latitude: 21.0, Longitude: 20.0, City: "ZGeo", PropertyStatus: property.StatusActive, CreatedBy: testUserID}

	idA, err := testRepo.Create(testCtx, pA)
	require.NoError(t, err)
	idB, err := testRepo.Create(testCtx, pB)
	require.NoError(t, err)

	props, _, err := testRepo.List(testCtx, property.ListRequest{Filter: property.Filter{Latitude: 0, Longitude: 0, RadiusKm: 50, City: "ZGeo"}, Limit: 50, Offset: 0})
	require.NoError(t, err)
	ids := make(map[int]bool)
	for _, p := range props {
		ids[p.ID] = true
	}
	assert.True(t, ids[idA])
	assert.True(t, ids[idB])

	props2, _, err := testRepo.List(testCtx, property.ListRequest{Filter: property.Filter{Latitude: 20.0, Longitude: 20.0, RadiusKm: -10, City: "ZGeo"}, Limit: 50, Offset: 0})
	require.NoError(t, err)
	ids2 := make(map[int]bool)
	for _, p := range props2 {
		ids2[p.ID] = true
	}
	assert.True(t, ids2[idA])
	assert.True(t, ids2[idB])
}

func TestCRUD_AdditionalCases(t *testing.T) {

	p := property.Property{
		Title:               "CRUD Test",
		PropertyDescription: "Verify fields",
		TypeID:              1,
		TransactionType:     property.TransactionSale,
		Price:               12345.67,
		Area:                45.5,
		PropertyAddress:     "AddrCrud",
		Latitude:            33.3,
		Longitude:           44.4,
		City:                "CrudCity",
		PropertyStatus:      property.StatusActive,
		CreatedBy:           testUserID,
	}
	id, err := testRepo.Create(testCtx, p)
	require.NoError(t, err)

	got, err := testRepo.GetByID(testCtx, id)
	require.NoError(t, err)
	assert.Equal(t, p.Title, got.Title)
	assert.Equal(t, p.Price, got.Price)

	_, err = testRepo.Update(testCtx, property.Property{ID: 9999999, Title: "x"})
	assert.Error(t, err)
	assert.True(t, basedb.IsNotFound(err))

	_, err = testRepo.Delete(testCtx, id)
	require.NoError(t, err)
	_, err = testRepo.Delete(testCtx, id)
	assert.Error(t, err)
	assert.True(t, basedb.IsNotFound(err))
}

func TestListPagination(t *testing.T) {
	city := "PaginateCity"

	created := make([]int, 0, 35)
	for i := 0; i < 35; i++ {
		p := property.Property{Title: fmt.Sprintf("Pg%03d", i), PropertyDescription: "P", TypeID: 1, TransactionType: property.TransactionSale, Price: float64(100 + i), Area: float64(10 + i), PropertyAddress: "Addr", Latitude: 1.0 + float64(i), Longitude: 1.0, City: city, PropertyStatus: property.StatusActive, CreatedBy: testUserID}
		id, err := testRepo.Create(testCtx, p)
		require.NoError(t, err)
		created = append(created, id)
	}

	perPage := 10
	gotIDs := make(map[int]bool)
	page := 0
	for {
		props, total, err := testRepo.List(testCtx, property.ListRequest{Filter: property.Filter{City: city}, Limit: perPage, Offset: page * perPage})
		require.NoError(t, err)
		if page == 0 {

			assert.GreaterOrEqual(t, total, len(created))
		}
		if len(props) == 0 {
			break
		}
		for _, p := range props {
			gotIDs[p.ID] = true
		}
		page++
		if page > 10 {
			break
		}
	}

	for _, id := range created {
		assert.True(t, gotIDs[id], "expected created id %d to appear in pages", id)
	}
}
