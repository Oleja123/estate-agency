package propertyservice

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	CreateFn  func(ctx context.Context, p domain.Property) (int, error)
	GetByIDFn func(ctx context.Context, id int) (domain.Property, error)
	UpdateFn  func(ctx context.Context, p domain.Property) error
	DeleteFn  func(ctx context.Context, id int) error
	ListFn    func(ctx context.Context, req domain.ListRequest) ([]domain.Property, error)
}

func (m *mockRepo) Create(ctx context.Context, p domain.Property) (int, error) {
	return m.CreateFn(ctx, p)
}
func (m *mockRepo) GetByID(ctx context.Context, id int) (domain.Property, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockRepo) Update(ctx context.Context, p domain.Property) error { return m.UpdateFn(ctx, p) }
func (m *mockRepo) Delete(ctx context.Context, id int) error            { return m.DeleteFn(ctx, id) }
func (m *mockRepo) List(ctx context.Context, req domain.ListRequest) ([]domain.Property, error) {
	return m.ListFn(ctx, req)
}

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCreateProperty_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, p domain.Property) (int, error) { return 10, nil }
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "x"}, nil
	}

	svc := New(repo, logger())
	got, err := svc.Create(ctx, dto.CreatePropertyRequest{Title: "x", CreatedBy: 1})
	require.NoError(t, err)
	assert.Equal(t, 10, got.ID)
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{}, dberrors.NewErrNotFound("property", id)
	}
	svc := New(repo, logger())
	_, err := svc.GetByID(ctx, 5)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestUpdateProperty_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "old"}, nil
	}
	repo.UpdateFn = func(ctx context.Context, p domain.Property) error { return nil }
	svc := New(repo, logger())
	err := svc.Update(ctx, dto.UpdatePropertyRequest{ID: 2, Title: "new"})
	require.NoError(t, err)
}

func TestListProperties_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]domain.Property, error) {
		return []domain.Property{{ID: 1}, {ID: 2}}, nil
	}
	svc := New(repo, logger())
	res, err := svc.List(ctx, dto.ListPropertiesRequest{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, res.Properties, 2)
	assert.Equal(t, 2, res.Total)
}

func TestDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.DeleteFn = func(ctx context.Context, id int) error { return dberrors.NewErrNotFound("property", id) }
	svc := New(repo, logger())
	err := svc.Delete(ctx, 7)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}
