package propertyservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"io"
	"log/slog"

	geocoder "github.com/Oleja123/estate-agency/internal/infrastructure/geocoder"

	optional "github.com/denpa16/optional-go-type"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favdto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	favdomain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
	ptypedomain "github.com/Oleja123/estate-agency/internal/domain/property_type"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFavoriteService struct{}

func (m *mockFavoriteService) Create(ctx context.Context, req favdto.CreateFavoriteRequest) (favdomain.Favorite, error) {
	return favdomain.Favorite{}, nil
}
func (m *mockFavoriteService) GetByUserAndProperty(ctx context.Context, key favdto.CreateFavoriteRequest) (favdomain.Favorite, error) {
	return favdomain.Favorite{}, nil
}
func (m *mockFavoriteService) Delete(ctx context.Context, key favdto.CreateFavoriteRequest) (int, error) {
	return 0, nil
}
func (m *mockFavoriteService) List(ctx context.Context, req favdto.ListFavoritesRequest) (favdto.ListFavoritesResponse, error) {
	return favdto.ListFavoritesResponse{}, nil
}
func (m *mockFavoriteService) Exists(ctx context.Context, key favdto.CreateFavoriteRequest) (bool, error) {
	return false, nil
}

type mockRepo struct {
	CreateFn  func(ctx context.Context, p domain.Property) (int, error)
	GetByIDFn func(ctx context.Context, id int) (domain.Property, error)
	UpdateFn  func(ctx context.Context, p domain.Property) error
	DeleteFn  func(ctx context.Context, id int) (int, error)
	ListFn    func(ctx context.Context, req domain.ListRequest) ([]domain.Property, int, error)
}

type mockTypeRepo struct {
	CreateFn    func(ctx context.Context, pt ptypedomain.PropertyType) (int, error)
	GetByIDFn   func(ctx context.Context, id int) (ptypedomain.PropertyType, error)
	GetByNameFn func(ctx context.Context, name string) (ptypedomain.PropertyType, error)
	UpdateFn    func(ctx context.Context, pt ptypedomain.PropertyType) error
	DeleteFn    func(ctx context.Context, id int) (int, error)
	ListFn      func(ctx context.Context, req ptypedomain.ListRequest) ([]ptypedomain.PropertyType, int, error)
}

func (m *mockTypeRepo) Create(ctx context.Context, pt ptypedomain.PropertyType) (int, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, pt)
	}
	return 0, nil
}

func (m *mockTypeRepo) GetByID(ctx context.Context, id int) (ptypedomain.PropertyType, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return ptypedomain.PropertyType{}, nil
}

func (m *mockTypeRepo) GetByName(ctx context.Context, name string) (ptypedomain.PropertyType, error) {
	if m.GetByNameFn != nil {
		return m.GetByNameFn(ctx, name)
	}
	return ptypedomain.PropertyType{}, nil
}

func (m *mockTypeRepo) Update(ctx context.Context, pt ptypedomain.PropertyType) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, pt)
	}
	return nil
}

func (m *mockTypeRepo) Delete(ctx context.Context, id int) (int, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return 0, nil
}

func (m *mockTypeRepo) List(ctx context.Context, req ptypedomain.ListRequest) ([]ptypedomain.PropertyType, int, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, req)
	}
	return nil, 0, nil
}

func (m *mockRepo) Create(ctx context.Context, p domain.Property) (int, error) {
	return m.CreateFn(ctx, p)
}
func (m *mockRepo) GetByID(ctx context.Context, id int) (domain.Property, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *mockRepo) Update(ctx context.Context, p domain.Property) error { return m.UpdateFn(ctx, p) }
func (m *mockRepo) Delete(ctx context.Context, id int) (int, error)     { return m.DeleteFn(ctx, id) }
func (m *mockRepo) List(ctx context.Context, req domain.ListRequest) ([]domain.Property, int, error) {
	return m.ListFn(ctx, req)
}

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCreateProperty_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, p domain.Property) (int, error) { return 10, nil }
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "x"}, nil
	}

	typeRepo := &mockTypeRepo{}
	typeRepo.GetByIDFn = func(ctx context.Context, id int) (ptypedomain.PropertyType, error) {
		return ptypedomain.PropertyType{Id: id, Name: "apartment"}, nil
	}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	got, err := svc.Create(ctx, 1, dto.CreatePropertyRequest{Title: "x", TypeID: 1, PropertyAddress: "addr"})
	require.NoError(t, err)
	assert.Equal(t, 10, got.ID)
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{}, dberrors.NewErrNotFound("property", id)
	}
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	_, err := svc.GetByID(ctx, 5)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestUpdateProperty_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "old", PropertyAddress: "addr"}, nil
	}
	repo.UpdateFn = func(ctx context.Context, p domain.Property) error { return nil }
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	title := "new"
	err := svc.Update(ctx, dto.UpdatePropertyRequest{ID: 2, Title: optional.OptionalString{Defined: true, Valid: true, Value: &title}})
	require.NoError(t, err)
}

func TestListProperties_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, req domain.ListRequest) ([]domain.Property, int, error) {
		return []domain.Property{{ID: 1}, {ID: 2}}, 2, nil
	}
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	res, err := svc.List(ctx, dto.ListPropertiesRequest{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, res.Properties, 2)
	assert.Equal(t, 2, res.Total)
}

func TestDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.DeleteFn = func(ctx context.Context, id int) (int, error) { return 0, dberrors.NewErrNotFound("property", id) }
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	_, err := svc.Delete(ctx, 7)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestCreateProperty_TypeNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}

	typeRepo := &mockTypeRepo{}
	typeRepo.GetByIDFn = func(ctx context.Context, id int) (ptypedomain.PropertyType, error) {
		return ptypedomain.PropertyType{}, dberrors.NewErrNotFound("property_type", id)
	}

	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	_, err := svc.Create(ctx, 1, dto.CreatePropertyRequest{Title: "x", TypeID: 99})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestUpdateProperty_TypeNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "old", TypeID: 1}, nil
	}
	typeRepo := &mockTypeRepo{}
	typeRepo.GetByIDFn = func(ctx context.Context, id int) (ptypedomain.PropertyType, error) {
		return ptypedomain.PropertyType{}, dberrors.NewErrNotFound("property_type", id)
	}

	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	tid := 99
	err := svc.Update(ctx, dto.UpdatePropertyRequest{ID: 2, TypeID: optional.OptionalInt{Defined: true, Valid: true, Value: &tid}})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestUpdateProperty_PartialPriceArea(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}

	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "old", Price: 100.0, Area: 50.0, TypeID: 1, PropertyAddress: "addr"}, nil
	}
	repo.UpdateFn = func(ctx context.Context, p domain.Property) error {

		if p.Price != 200.5 {
			return fmt.Errorf("price not updated: %v", p.Price)
		}
		if p.Area != 75.25 {
			return fmt.Errorf("area not updated: %v", p.Area)
		}
		if p.Title != "old" {
			return fmt.Errorf("title was modified: %s", p.Title)
		}
		if p.TypeID != 1 {
			return fmt.Errorf("type id was modified: %d", p.TypeID)
		}
		return nil
	}
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})

	price := 200.5
	area := 75.25
	err := svc.Update(ctx, dto.UpdatePropertyRequest{ID: 2, Price: optional.OptionalFloat64{Defined: true, Valid: true, Value: &price}, Area: optional.OptionalFloat64{Defined: true, Valid: true, Value: &area}})
	require.NoError(t, err)
}

func TestCreateProperty_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, p domain.Property) (int, error) {
		return 0, dberrors.NewErrAlreadyExists("property", "title", p.Title)
	}
	typeRepo := &mockTypeRepo{}
	typeRepo.GetByIDFn = func(ctx context.Context, id int) (ptypedomain.PropertyType, error) {
		return ptypedomain.PropertyType{Id: id, Name: "apartment"}, nil
	}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	_, err := svc.Create(ctx, 1, dto.CreatePropertyRequest{Title: "x", TypeID: 1, PropertyAddress: "addr"})
	require.Error(t, err)
	var ae apperrors.ErrAlreadyExists
	assert.True(t, errors.As(err, &ae))
}

func TestUpdateProperty_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "old", PropertyAddress: "addr"}, nil
	}
	repo.UpdateFn = func(ctx context.Context, p domain.Property) error {
		return dberrors.NewErrAlreadyExists("property", "title", p.Title)
	}
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), geocoder.NewNoop(), &mockFavoriteService{})
	title := "x"
	err := svc.Update(ctx, dto.UpdatePropertyRequest{ID: 2, Title: optional.OptionalString{Defined: true, Valid: true, Value: &title}})
	require.Error(t, err)
	var ae apperrors.ErrAlreadyExists
	assert.True(t, errors.As(err, &ae))
}

type mockGeo struct {
	GeocodeFn func(address string) (float64, float64, error)
}

func (m *mockGeo) Geocode(address string) (float64, float64, error) {
	if m.GeocodeFn != nil {
		return m.GeocodeFn(address)
	}
	return 0, 0, nil
}

func TestCreateProperty_EmptyAddress(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	typeRepo := &mockTypeRepo{}
	typeRepo.GetByIDFn = func(ctx context.Context, id int) (ptypedomain.PropertyType, error) {
		return ptypedomain.PropertyType{Id: id, Name: "apartment"}, nil
	}
	svc := New(repo, typeRepo, logger(), &mockGeo{}, &mockFavoriteService{})
	_, err := svc.Create(ctx, 1, dto.CreatePropertyRequest{Title: "x", TypeID: 1, PropertyAddress: ""})
	require.Error(t, err)
	var di apperrors.ErrInvalidInput
	assert.True(t, errors.As(err, &di))
}

func TestCreateProperty_GeocodeError(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, p domain.Property) (int, error) { return 10, nil }
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "x"}, nil
	}
	typeRepo := &mockTypeRepo{}
	typeRepo.GetByIDFn = func(ctx context.Context, id int) (ptypedomain.PropertyType, error) {
		return ptypedomain.PropertyType{Id: id, Name: "apartment"}, nil
	}
	svc := New(repo, typeRepo, logger(), &mockGeo{GeocodeFn: func(address string) (float64, float64, error) {
		return 0, 0, fmt.Errorf("geo fail")
	}}, &mockFavoriteService{})

	_, err := svc.Create(ctx, 1, dto.CreatePropertyRequest{Title: "x", TypeID: 1, PropertyAddress: "addr"})
	require.Error(t, err)
	var ge apperrors.ErrGeocoding
	assert.True(t, errors.As(err, &ge))
}

func TestUpdateProperty_SetEmptyAddress(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "old", PropertyAddress: "addr"}, nil
	}
	repo.UpdateFn = func(ctx context.Context, p domain.Property) error { return nil }
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), &mockGeo{}, &mockFavoriteService{})

	empty := ""
	req := dto.UpdatePropertyRequest{ID: 2, PropertyAddress: optional.OptionalString{Defined: true, Valid: true, Value: &empty}}
	err := svc.Update(ctx, req)
	require.Error(t, err)
	var di apperrors.ErrInvalidInput
	assert.True(t, errors.As(err, &di))
}

func TestUpdateProperty_GeocodeErrorOnUpdate(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{ID: id, Title: "old", PropertyAddress: "addr"}, nil
	}
	repo.UpdateFn = func(ctx context.Context, p domain.Property) error { return nil }
	typeRepo := &mockTypeRepo{}
	svc := New(repo, typeRepo, logger(), &mockGeo{GeocodeFn: func(address string) (float64, float64, error) {
		return 0, 0, fmt.Errorf("geo fail")
	}}, &mockFavoriteService{})

	addr := "new addr"
	req := dto.UpdatePropertyRequest{ID: 2, PropertyAddress: optional.OptionalString{Defined: true, Valid: true, Value: &addr}}
	err := svc.Update(ctx, req)
	require.Error(t, err)
	var ge apperrors.ErrGeocoding
	assert.True(t, errors.As(err, &ge))
}
