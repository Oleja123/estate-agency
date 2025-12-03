package imageservice

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	"io"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/image"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	filestore "github.com/Oleja123/estate-agency/internal/infrastructure/filestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	CreateFn     func(ctx context.Context, img domain.PropertyImage) (int, error)
	CreateManyFn func(ctx context.Context, imgs []domain.PropertyImage) ([]int, error)
	DeleteManyFn func(ctx context.Context, propertyID int) ([]int, error)
	GetByIDFn    func(ctx context.Context, id int) (domain.PropertyImage, error)
	ListFn       func(ctx context.Context, propertyID int) ([]domain.PropertyImage, error)
	DeleteFn     func(ctx context.Context, id int) (int, error)
}

func (m *mockRepo) Create(ctx context.Context, img domain.PropertyImage) (int, error) {
	if m.CreateFn == nil {
		return 0, nil
	}
	return m.CreateFn(ctx, img)
}
func (m *mockRepo) CreateMany(ctx context.Context, imgs []domain.PropertyImage) ([]int, error) {
	if m.CreateManyFn == nil {
		return nil, nil
	}
	return m.CreateManyFn(ctx, imgs)
}
func (m *mockRepo) DeleteMany(ctx context.Context, propertyID int) ([]int, error) {
	if m.DeleteManyFn == nil {
		return nil, nil
	}
	return m.DeleteManyFn(ctx, propertyID)
}
func (m *mockRepo) GetByID(ctx context.Context, id int) (domain.PropertyImage, error) {
	if m.GetByIDFn == nil {
		return domain.PropertyImage{}, nil
	}
	return m.GetByIDFn(ctx, id)
}
func (m *mockRepo) ListByProperty(ctx context.Context, propertyID int) ([]domain.PropertyImage, error) {
	if m.ListFn == nil {
		return nil, nil
	}
	return m.ListFn(ctx, propertyID)
}
func (m *mockRepo) Delete(ctx context.Context, id int) (int, error) {
	if m.DeleteFn == nil {
		return 0, nil
	}
	return m.DeleteFn(ctx, id)
}

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCreate_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateFn = func(ctx context.Context, img domain.PropertyImage) (int, error) { return 11, nil }
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.PropertyImage, error) {
		return domain.PropertyImage{ID: id, PropertyID: 5, Path: "/tmp/x.jpg"}, nil
	}

	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())

	jpeg := []byte{0xFF, 0xD8, 0xFF}
	got, err := svc.Create(ctx, dto.CreateImageRequest{PropertyID: 5, File: dto.ImageFile{Filename: "x.jpg", Data: jpeg}})
	require.NoError(t, err)
	assert.Equal(t, 11, got.ID)
}

func TestCreate_InvalidInput(t *testing.T) {
	ctx := context.Background()
	store := filestore.New(t.TempDir())
	svc := New(&mockRepo{}, logger(), store, t.TempDir())
	_, err := svc.Create(ctx, dto.CreateImageRequest{PropertyID: 0, File: dto.ImageFile{Filename: "", Data: nil}})
	require.Error(t, err)
}

func TestCreateMany_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.CreateManyFn = func(ctx context.Context, imgs []domain.PropertyImage) ([]int, error) { return []int{1, 2, 3}, nil }
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.PropertyImage, error) {
		return domain.PropertyImage{ID: id, PropertyID: 7, Path: "/tmp/x.jpg"}, nil
	}

	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())
	jpeg := []byte{0xFF, 0xD8, 0xFF}
	items := []dto.ImageFile{{Filename: "a.jpg", Data: jpeg}, {Filename: "b.jpg", Data: jpeg}, {Filename: "c.jpg", Data: jpeg}}
	res, err := svc.CreateMany(ctx, dto.CreateImagesRequest{PropertyID: 7, Files: items})
	require.NoError(t, err)
	assert.Len(t, res, 3)
}

func TestCreateMany_Empty(t *testing.T) {
	ctx := context.Background()
	store := filestore.New(t.TempDir())
	svc := New(&mockRepo{}, logger(), store, t.TempDir())
	res, err := svc.CreateMany(ctx, dto.CreateImagesRequest{PropertyID: 1, Files: nil})
	require.NoError(t, err)
	assert.Nil(t, res)
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.GetByIDFn = func(ctx context.Context, id int) (domain.PropertyImage, error) {
		return domain.PropertyImage{}, dberrors.NewErrNotFound("property_image", id)
	}
	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())
	_, err := svc.GetByID(ctx, 9)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.DeleteFn = func(ctx context.Context, id int) (int, error) {
		return 0, dberrors.NewErrNotFound("property_image", id)
	}
	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())
	_, err := svc.Delete(ctx, 3)
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestCreate_PropertyNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, propertyID int) ([]domain.PropertyImage, error) { return nil, nil }
	repo.DeleteManyFn = func(ctx context.Context, propertyID int) ([]int, error) { return nil, nil }
	repo.CreateFn = func(ctx context.Context, img domain.PropertyImage) (int, error) {
		return 0, dberrors.NewErrForeignKeyViolation("property", "fk_property", "property_id")
	}
	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())
	jpeg := []byte{0xFF, 0xD8, 0xFF}
	_, err := svc.Create(ctx, dto.CreateImageRequest{PropertyID: 123, File: dto.ImageFile{Filename: "x.jpg", Data: jpeg}})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}

func TestCreate_InvalidFileFormat(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	// repo behaviors for create will not be reached because filestore will reject
	repo.ListFn = func(ctx context.Context, propertyID int) ([]domain.PropertyImage, error) { return nil, nil }
	repo.DeleteManyFn = func(ctx context.Context, propertyID int) ([]int, error) { return nil, nil }
	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())

	// send random bytes with .txt extension -> unsupported format
	bad := []byte("not an image")
	_, err := svc.Create(ctx, dto.CreateImageRequest{PropertyID: 1, File: dto.ImageFile{Filename: "bad.txt", Data: bad}})
	require.Error(t, err)
	var inval apperrors.ErrInvalidInput
	assert.True(t, errors.As(err, &inval))
}

func TestCreateMany_EmptyFileInList(t *testing.T) {
	ctx := context.Background()
	store := filestore.New(t.TempDir())
	svc := New(&mockRepo{}, logger(), store, t.TempDir())
	jpeg := []byte{0xFF, 0xD8, 0xFF}
	items := []dto.ImageFile{{Filename: "ok.jpg", Data: jpeg}, {Filename: "", Data: nil}}
	_, err := svc.CreateMany(ctx, dto.CreateImagesRequest{PropertyID: 7, Files: items})
	require.Error(t, err)
	var inval apperrors.ErrInvalidInput
	assert.True(t, errors.As(err, &inval))
}

func TestCreateWhenDeleteManyFails(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	// simulate existing images so service will call DeleteMany
	repo.ListFn = func(ctx context.Context, propertyID int) ([]domain.PropertyImage, error) {
		return []domain.PropertyImage{{ID: 1, PropertyID: propertyID, Path: "/x"}}, nil
	}
	repo.DeleteManyFn = func(ctx context.Context, propertyID int) ([]int, error) { return nil, errors.New("db failure") }
	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())

	jpeg := []byte{0xFF, 0xD8, 0xFF}
	_, err := svc.Create(ctx, dto.CreateImageRequest{PropertyID: 5, File: dto.ImageFile{Filename: "x.jpg", Data: jpeg}})
	require.Error(t, err)
	// expect internal error
	var ie apperrors.ErrInternal
	assert.True(t, errors.As(err, &ie))
}

func TestCreateMany_TooManyFiles(t *testing.T) {
	ctx := context.Background()
	store := filestore.New(t.TempDir())
	svc := New(&mockRepo{}, logger(), store, t.TempDir())
	files := make([]dto.ImageFile, 0, 120)
	for i := 0; i < 111; i++ {
		files = append(files, dto.ImageFile{Filename: "a.jpg", Data: []byte{0xFF, 0xD8, 0xFF}})
	}
	_, err := svc.CreateMany(ctx, dto.CreateImagesRequest{PropertyID: 1, Files: files})
	require.Error(t, err)
	var inval apperrors.ErrInvalidInput
	assert.True(t, errors.As(err, &inval))
}

func TestCreateMany_PropertyNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	repo.ListFn = func(ctx context.Context, propertyID int) ([]domain.PropertyImage, error) { return nil, nil }
	repo.DeleteManyFn = func(ctx context.Context, propertyID int) ([]int, error) { return nil, nil }
	repo.CreateManyFn = func(ctx context.Context, imgs []domain.PropertyImage) ([]int, error) {
		return nil, dberrors.NewErrForeignKeyViolation("property", "fk_property", "property_id")
	}
	store := filestore.New(t.TempDir())
	svc := New(repo, logger(), store, t.TempDir())
	jpeg := []byte{0xFF, 0xD8, 0xFF}
	_, err := svc.CreateMany(ctx, dto.CreateImagesRequest{PropertyID: 123, Files: []dto.ImageFile{{Filename: "a.jpg", Data: jpeg}}})
	require.Error(t, err)
	var nf apperrors.ErrNotFound
	assert.True(t, errors.As(err, &nf))
}
