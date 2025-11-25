package propertytypeservice

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property_type"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
)

type mockRepo struct {
	CreateFn    func(ctx context.Context, pt domain.PropertyType) (int, error)
	GetByIDFn   func(ctx context.Context, id int) (domain.PropertyType, error)
	GetByNameFn func(ctx context.Context, name string) (domain.PropertyType, error)
	UpdateFn    func(ctx context.Context, pt domain.PropertyType) error
	DeleteFn    func(ctx context.Context, id int) error
	ListFn      func(ctx context.Context, req domain.ListRequest) ([]domain.PropertyType, error)
}

func (m *mockRepo) Create(ctx context.Context, pt domain.PropertyType) (int, error) {
	return m.CreateFn(ctx, pt)
}
func (m *mockRepo) GetByID(ctx context.Context, id int) (domain.PropertyType, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockRepo) GetByName(ctx context.Context, name string) (domain.PropertyType, error) {
	return m.GetByNameFn(ctx, name)
}
func (m *mockRepo) Update(ctx context.Context, pt domain.PropertyType) error {
	return m.UpdateFn(ctx, pt)
}
func (m *mockRepo) Delete(ctx context.Context, id int) error { return m.DeleteFn(ctx, id) }
func (m *mockRepo) List(ctx context.Context, req domain.ListRequest) ([]domain.PropertyType, error) {
	return m.ListFn(ctx, req)
}

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCreate_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, pt domain.PropertyType) (int, error) { return 1, nil }
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.PropertyType, error) {
		return domain.PropertyType{Id: id, Name: "apartment"}, nil
	}

	svc := New(repo, logger())
	got, err := svc.Create(ctx, dto.CreatePropertyTypeRequest{Name: "apartment"})
	require.NoError(t, err)
	assert.Equal(t, "apartment", got.Name)
	assert.Equal(t, 1, got.Id)
}

func TestCreate_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, pt domain.PropertyType) (int, error) {
		return 0, dberrors.NewErrAlreadyExists("property_type", "name", pt.Name)
	}

	svc := New(repo, logger())
	_, err := svc.Create(ctx, dto.CreatePropertyTypeRequest{Name: "apartment"})
	require.Error(t, err)
	var ae apperrors.ErrAlreadyExists
	assert.True(t, errors.As(err, &ae))
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.PropertyType, error) {
		return domain.PropertyType{}, dberrors.NewErrNotFound("property_type", id)
	}
	svc := New(repo, logger())
	_, err := svc.GetByID(ctx, 42)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestUpdate_Success(t *testing.T) {
	ctx := context.Background()
	existing := domain.PropertyType{Id: 2, Name: "house"}
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.PropertyType, error) { return existing, nil }
	repo.UpdateFn = func(ctx context.Context, pt domain.PropertyType) error { return nil }

	svc := New(repo, logger())
	err := svc.Update(ctx, dto.UpdatePropertyTypeRequest{ID: 2, Name: "villa"})
	require.NoError(t, err)
}

func TestUpdate_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.PropertyType, error) {
		return domain.PropertyType{}, dberrors.NewErrNotFound("property_type", id)
	}
	svc := New(repo, logger())
	err := svc.Update(ctx, dto.UpdatePropertyTypeRequest{ID: 99, Name: "x"})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestList_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]domain.PropertyType, error) {
		return []domain.PropertyType{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}, nil
	}
	svc := New(repo, logger())
	res, err := svc.List(ctx, dto.ListPropertyTypesRequest{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, res.Types, 2)
	assert.Equal(t, 2, res.Total)
}

func TestDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.DeleteFn = func(ctx context.Context, id int) error {
		return dberrors.NewErrNotFound("property_type", id)
	}
	svc := New(repo, logger())
	err := svc.Delete(ctx, 5)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}
