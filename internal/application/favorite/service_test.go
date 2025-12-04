package favoriteservice

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	imagedomain "github.com/Oleja123/estate-agency/internal/domain/image"
	prop "github.com/Oleja123/estate-agency/internal/domain/property"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/Oleja123/estate-agency/internal/infrastructure/filestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	CreateFn               func(ctx context.Context, fav domain.Favorite) error
	GetByUserAndPropertyFn func(ctx context.Context, userID, propertyID int) (prop.Property, error)
	DeleteFn               func(ctx context.Context, userID, propertyID int) (int, error)
	ListFn                 func(ctx context.Context, req domain.ListRequest) ([]prop.Property, int, error)
	ExistsFn               func(ctx context.Context, userID, propertyID int) (bool, error)
}

func (m *mockRepo) Create(ctx context.Context, fav domain.Favorite) error {
	if m.CreateFn == nil {
		return nil
	}
	return m.CreateFn(ctx, fav)
}
func (m *mockRepo) GetByUserAndProperty(ctx context.Context, userID, propertyID int) (prop.Property, error) {
	if m.GetByUserAndPropertyFn == nil {
		return prop.Property{}, nil
	}
	return m.GetByUserAndPropertyFn(ctx, userID, propertyID)
}
func (m *mockRepo) Delete(ctx context.Context, userID, propertyID int) (int, error) {
	if m.DeleteFn == nil {
		return 0, nil
	}
	return m.DeleteFn(ctx, userID, propertyID)
}
func (m *mockRepo) List(ctx context.Context, req domain.ListRequest) ([]prop.Property, int, error) {
	if m.ListFn == nil {
		return nil, 0, nil
	}
	return m.ListFn(ctx, req)
}
func (m *mockRepo) Exists(ctx context.Context, userID, propertyID int) (bool, error) {
	if m.ExistsFn == nil {
		return false, nil
	}
	return m.ExistsFn(ctx, userID, propertyID)
}

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCreate_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, fav domain.Favorite) error { return nil }
	repo.GetByUserAndPropertyFn = func(ctx context.Context, userID, propertyID int) (prop.Property, error) {
		return prop.Property{ID: propertyID, CreatedBy: userID}, nil
	}

	svc := New(repo, logger(), nil)
	got, err := svc.Create(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, got)
}

func TestCreate_InvalidInput(t *testing.T) {
	ctx := context.Background()
	svc := New(&mockRepo{}, logger(), nil)
	_, err := svc.Create(ctx, dto.CreateFavoriteRequest{UserID: 0, PropertyID: 0})
	require.Error(t, err)
	var ie apperrors.ErrInvalidInput
	assert.True(t, errors.As(err, &ie))
}

func TestGetByUserAndProperty_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByUserAndPropertyFn = func(ctx context.Context, userID, propertyID int) (prop.Property, error) {
		return prop.Property{}, dberrors.NewErrNotFound("favorite", nil)
	}
	svc := New(repo, logger(), nil)
	_, err := svc.GetByUserAndProperty(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.DeleteFn = func(ctx context.Context, userID, propertyID int) (int, error) {
		return 0, dberrors.NewErrNotFound("favorite", nil)
	}
	svc := New(repo, logger(), nil)
	_, err := svc.Delete(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestList_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]prop.Property, int, error) {
		return []prop.Property{{ID: 2}, {ID: 3}}, 2, nil
	}
	svc := New(repo, logger(), nil)
	res, err := svc.List(ctx, dto.ListFavoritesRequest{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, res.Favorites, 2)
}

func TestExists_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ExistsFn = func(ctx context.Context, userID, propertyID int) (bool, error) { return true, nil }
	svc := New(repo, logger(), nil)
	ok, err := svc.Exists(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestList_WithNilFileStore_ReturnsFilenamesEmptyData(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]prop.Property, int, error) {
		return []prop.Property{{ID: 2, Images: []imagedomain.PropertyImage{{Path: "/tmp/some/path/img1.jpg"}}}}, 1, nil
	}
	svc := New(repo, logger(), nil)
	res, err := svc.List(ctx, dto.ListFavoritesRequest{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Len(t, res.Favorites, 1)
	f := res.Favorites[0]
	require.Len(t, f.Images, 1)
	assert.Equal(t, "img1.jpg", f.Images[0].Filename)
	assert.Nil(t, f.Images[0].Data)
}

func TestList_WithFileStore_ReadsData(t *testing.T) {
	ctx := context.Background()

	td := t.TempDir()
	imgPath := filepath.Join(td, "imgA.jpg")
	content := []byte("helloimg")
	require.NoError(t, os.WriteFile(imgPath, content, 0o644))

	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]prop.Property, int, error) {
		return []prop.Property{{ID: 2, Images: []imagedomain.PropertyImage{{Path: imgPath}}}}, 1, nil
	}
	fs := filestore.New("")
	svc := New(repo, logger(), fs)
	res, err := svc.List(ctx, dto.ListFavoritesRequest{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Len(t, res.Favorites, 1)
	f := res.Favorites[0]
	require.Len(t, f.Images, 1)
	assert.Equal(t, "imgA.jpg", f.Images[0].Filename)
	require.NotNil(t, f.Images[0].Data)
	assert.Equal(t, content, f.Images[0].Data)

	require.NotNil(t, f.Image)
	assert.Equal(t, "imgA.jpg", f.Image.Filename)
}

func TestList_FileStoreReadError_ReturnsInternal(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}

	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]prop.Property, int, error) {
		return []prop.Property{{ID: 2, Images: []imagedomain.PropertyImage{{Path: ""}}}}, 1, nil
	}
	fs := filestore.New("")
	svc := New(repo, logger(), fs)
	_, err := svc.List(ctx, dto.ListFavoritesRequest{Limit: 10, Offset: 0})
	require.Error(t, err)
	var ie apperrors.ErrInternal
	assert.True(t, errors.As(err, &ie))
}
