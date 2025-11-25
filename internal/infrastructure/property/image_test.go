package propertydb

import (
	"context"
	"testing"

	"github.com/Oleja123/estate-agency/internal/domain/image"
	"github.com/Oleja123/estate-agency/internal/infrastructure/basedb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test file relies on the package-level TestMain and helpers
// from property_test.go which set up test DB, client and default data.

func createTestProperty(t *testing.T) int {
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

		imageRepo := NewImageRepository(testClient, testLogger)

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

		err = imageRepo.Delete(testCtx, id)
		require.NoError(t, err)

		_, err = imageRepo.GetByID(testCtx, id)
		assert.Error(t, err)
		assert.True(t, basedb.IsNotFound(err))
	})
}
