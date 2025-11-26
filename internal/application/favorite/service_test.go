package favoriteservice

import (
	"context"
	"errors"
	"io"
	"testing"

	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	CreateFn               func(ctx context.Context, fav domain.Favorite) error
	GetByUserAndPropertyFn func(ctx context.Context, userID, propertyID int) (domain.Favorite, error)
	DeleteFn               func(ctx context.Context, userID, propertyID int) error
	ListFn                 func(ctx context.Context, req domain.ListRequest) ([]domain.Favorite, error)
	ExistsFn               func(ctx context.Context, userID, propertyID int) (bool, error)
}

func (m *mockRepo) Create(ctx context.Context, fav domain.Favorite) error {
	if m.CreateFn == nil {
		return nil
	}
	return m.CreateFn(ctx, fav)
}
func (m *mockRepo) GetByUserAndProperty(ctx context.Context, userID, propertyID int) (domain.Favorite, error) {
	if m.GetByUserAndPropertyFn == nil {
		return domain.Favorite{}, nil
	}
	return m.GetByUserAndPropertyFn(ctx, userID, propertyID)
}
func (m *mockRepo) Delete(ctx context.Context, userID, propertyID int) error {
	if m.DeleteFn == nil {
		return nil
	}
	return m.DeleteFn(ctx, userID, propertyID)
}
func (m *mockRepo) List(ctx context.Context, req domain.ListRequest) ([]domain.Favorite, error) {
	if m.ListFn == nil {
		return nil, nil
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
	repo.GetByUserAndPropertyFn = func(ctx context.Context, userID, propertyID int) (domain.Favorite, error) {
		return domain.Favorite{UserID: userID, PropertyID: propertyID}, nil
	}

	svc := New(repo, logger())
	got, err := svc.Create(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.NoError(t, err)
	assert.Equal(t, 1, got.UserID)
}

func TestCreate_InvalidInput(t *testing.T) {
	ctx := context.Background()
	svc := New(&mockRepo{}, logger())
	_, err := svc.Create(ctx, dto.CreateFavoriteRequest{UserID: 0, PropertyID: 0})
	require.Error(t, err)
	var ie apperrors.ErrInvalidInput
	assert.True(t, errors.As(err, &ie))
}

func TestGetByUserAndProperty_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByUserAndPropertyFn = func(ctx context.Context, userID, propertyID int) (domain.Favorite, error) {
		return domain.Favorite{}, dberrors.NewErrNotFound("favorite", nil)
	}
	svc := New(repo, logger())
	_, err := svc.GetByUserAndProperty(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.DeleteFn = func(ctx context.Context, userID, propertyID int) error {
		return dberrors.NewErrNotFound("favorite", nil)
	}
	svc := New(repo, logger())
	err := svc.Delete(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestList_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]domain.Favorite, error) {
		return []domain.Favorite{{UserID: 1, PropertyID: 2}, {UserID: 2, PropertyID: 3}}, nil
	}
	svc := New(repo, logger())
	res, err := svc.List(ctx, dto.ListFavoritesRequest{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, res.Favorites, 2)
}

func TestExists_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ExistsFn = func(ctx context.Context, userID, propertyID int) (bool, error) { return true, nil }
	svc := New(repo, logger())
	ok, err := svc.Exists(ctx, dto.CreateFavoriteRequest{UserID: 1, PropertyID: 2})
	require.NoError(t, err)
	assert.True(t, ok)
}
