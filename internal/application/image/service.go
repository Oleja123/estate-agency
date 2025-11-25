package imageservice

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "time"

    apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
    dto "github.com/Oleja123/estate-agency/internal/application/image/dto"
    domain "github.com/Oleja123/estate-agency/internal/domain/image"
    dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
)

var _ Service = (*service)(nil)

type service struct {
    repo     domain.ImageRepository
    logger   *slog.Logger
    basePath string // base directory where images are stored
}

// New constructs the image service. basePath is the root directory where images will be saved
// (e.g. "property_images"). Use an absolute or relative path. Tests may pass t.TempDir().
func New(repo domain.ImageRepository, logger *slog.Logger, basePath string) Service {
    if basePath == "" {
        basePath = "property_images"
    }
    return &service{repo: repo, logger: logger, basePath: basePath}
}

func (s *service) Create(ctx context.Context, req dto.CreateImageRequest) (domain.PropertyImage, error) {
    if req.PropertyID == 0 {
        return domain.PropertyImage{}, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "must be provided")
    }
    if req.File.Filename == "" || len(req.File.Data) == 0 {
        return domain.PropertyImage{}, apperrors.NewErrInvalidInput("file", nil, "must be provided")
    }

    // ensure directory exists
    dir := filepath.Join(s.basePath, fmt.Sprintf("%d", req.PropertyID))
    if err := os.MkdirAll(dir, 0o755); err != nil {
        s.logger.Error("create image: mkdir failed", "err", err)
        return domain.PropertyImage{}, apperrors.NewErrInternal("failed to save image")
    }

    // unique filename
    fname := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(req.File.Filename))
    fullpath := filepath.Join(dir, fname)
    if err := os.WriteFile(fullpath, req.File.Data, 0o644); err != nil {
        s.logger.Error("create image: write file failed", "err", err)
        return domain.PropertyImage{}, apperrors.NewErrInternal("failed to save image")
    }

    img := domain.PropertyImage{PropertyID: req.PropertyID, Path: fullpath}
    id, err := s.repo.Create(ctx, img)
    if err != nil {
        s.logger.Error("create image: repo create failed", "err", err)
        return domain.PropertyImage{}, apperrors.NewErrInternal("failed to create image")
    }

    created, err := s.repo.GetByID(ctx, id)
    if err != nil {
        s.logger.Error("create image: fetch created failed", "err", err)
        return domain.PropertyImage{}, apperrors.NewErrInternal("failed to fetch created image")
    }
    return created, nil
}

func (s *service) CreateMany(ctx context.Context, req dto.CreateImagesRequest) ([]domain.PropertyImage, error) {
    if req.PropertyID == 0 {
        return nil, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "must be provided")
    }
    if len(req.Files) == 0 {
        return nil, nil
    }

    // ensure directory exists
    dir := filepath.Join(s.basePath, fmt.Sprintf("%d", req.PropertyID))
    if err := os.MkdirAll(dir, 0o755); err != nil {
        s.logger.Error("create many: mkdir failed", "err", err)
        return nil, apperrors.NewErrInternal("failed to save images")
    }

    imgs := make([]domain.PropertyImage, 0, len(req.Files))
    for i, f := range req.Files {
        if f.Filename == "" || len(f.Data) == 0 {
            return nil, apperrors.NewErrInvalidInput("file", nil, "must be provided")
        }
        fname := fmt.Sprintf("%d_%d_%s", time.Now().UnixNano(), i, filepath.Base(f.Filename))
        fullpath := filepath.Join(dir, fname)
        if err := os.WriteFile(fullpath, f.Data, 0o644); err != nil {
            s.logger.Error("create many: write file failed", "err", err)
            return nil, apperrors.NewErrInternal("failed to save images")
        }
        imgs = append(imgs, domain.PropertyImage{PropertyID: req.PropertyID, Path: fullpath})
    }

    ids, err := s.repo.CreateMany(ctx, imgs)
    if err != nil {
        s.logger.Error("create many images: repo failed", "err", err)
        return nil, apperrors.NewErrInternal("failed to create images")
    }

    // fetch created images
    var created []domain.PropertyImage
    for _, id := range ids {
        img, err := s.repo.GetByID(ctx, id)
        if err != nil {
            s.logger.Error("create many: failed to fetch created image", "err", err, "id", id)
            return nil, apperrors.NewErrInternal("failed to fetch created images")
        }
        created = append(created, img)
    }
    return created, nil
}

func (s *service) GetByID(ctx context.Context, id int) (domain.PropertyImage, error) {
    img, err := s.repo.GetByID(ctx, id)
    if err != nil {
        var nf dberrors.ErrNotFound
        if errors.As(err, &nf) {
            return domain.PropertyImage{}, apperrors.NewErrNotFound("property_image", id)
        }
        s.logger.Error("get image: repo error", "err", err)
        return domain.PropertyImage{}, apperrors.NewErrInternal("failed to fetch image")
    }
    return img, nil
}

func (s *service) ListByProperty(ctx context.Context, propertyID int) ([]domain.PropertyImage, error) {
    list, err := s.repo.ListByProperty(ctx, propertyID)
    if err != nil {
        s.logger.Error("list images: repo failed", "err", err)
        return nil, apperrors.NewErrInternal("failed to list images")
    }
    return list, nil
}

func (s *service) Delete(ctx context.Context, id int) error {
    if err := s.repo.Delete(ctx, id); err != nil {
        var nf dberrors.ErrNotFound
        if errors.As(err, &nf) {
            return apperrors.NewErrNotFound("property_image", id)
        }
        s.logger.Error("delete image: repo failed", "err", err)
        return apperrors.NewErrInternal("failed to delete image")
    }
    return nil
}
