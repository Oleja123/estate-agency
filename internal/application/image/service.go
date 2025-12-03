package imageservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/image"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	filestore "github.com/Oleja123/estate-agency/internal/infrastructure/filestore"
)

var _ Service = (*service)(nil)

type service struct {
	repo    domain.ImageRepository
	logger  *slog.Logger
	store   *filestore.FileStore
	baseDir string
	locks   sync.Map
}

func (s *service) lockForProperty(propertyID int) func() {
	if propertyID == 0 {

		return func() {}
	}
	v, _ := s.locks.LoadOrStore(propertyID, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return func() {
		m.Unlock()
	}
}

func New(repo domain.ImageRepository, logger *slog.Logger, store *filestore.FileStore, basePath string) Service {
	if basePath == "" {
		basePath = "property_images"
	}
	return &service{repo: repo, logger: logger, store: store, baseDir: basePath}
}

func (s *service) Create(ctx context.Context, req dto.CreateImageRequest) (domain.PropertyImage, error) {
	if req.PropertyID == 0 {
		return domain.PropertyImage{}, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "must be provided")
	}
	if req.File.Filename == "" || len(req.File.Data) == 0 {
		return domain.PropertyImage{}, apperrors.NewErrInvalidInput("file", nil, "must be provided")
	}

	unlock := s.lockForProperty(req.PropertyID)
	defer unlock()

	existing, err := s.repo.ListByProperty(ctx, req.PropertyID)
	if err != nil {
		s.logger.Error("create image: list existing failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to prepare images")
	}
	if len(existing) > 0 {
		if _, err := s.repo.DeleteMany(ctx, req.PropertyID); err != nil {

			var nf dberrors.ErrNotFound
			if !errors.As(err, &nf) {
				s.logger.Error("create image: delete existing db rows failed", "err", err)
				return domain.PropertyImage{}, apperrors.NewErrInternal("failed to remove existing images")
			}
		}
		if delErr := s.store.DeletePropertyDir(req.PropertyID); delErr != nil {
			s.logger.Error("create image: filestore delete dir failed", "err", delErr, "property_id", req.PropertyID)
			return domain.PropertyImage{}, apperrors.NewErrInternal("failed to remove existing images")
		}
	}

	tmpDir, err := os.MkdirTemp(s.baseDir, fmt.Sprintf("%d_tmp_", req.PropertyID))
	if err != nil {
		s.logger.Error("create image: tmp dir failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to prepare storage")
	}

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	fname := fmt.Sprintf("%d%s", 1, filepath.Ext(req.File.Filename))

	fullTmpPath, err := s.store.SaveToDir(tmpDir, fname, req.File.Data)
	if err != nil {
		return domain.PropertyImage{}, s.mapFilestoreSaveError(err)
	}

	finalDir := filepath.Join(s.baseDir, fmt.Sprintf("%d", req.PropertyID))

	if err := s.store.DeletePropertyDir(req.PropertyID); err != nil {
		s.logger.Error("create image: remove final dir failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to prepare storage")
	}

	if err := os.Rename(tmpDir, finalDir); err != nil {
		s.logger.Error("create image: move temp to final failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to store image files")
	}

	finalPath := filepath.Join(finalDir, filepath.Base(fullTmpPath))

	img := domain.PropertyImage{PropertyID: req.PropertyID, Path: finalPath}
	id, err := s.repo.Create(ctx, img)
	if err != nil {
		var fk dberrors.ErrForeignKeyViolation
		var di dberrors.ErrInvalidInput
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &fk):

			return domain.PropertyImage{}, apperrors.NewErrNotFound("property", req.PropertyID)
		case errors.As(err, &di):
			return domain.PropertyImage{}, apperrors.NewErrInvalidInput(di.Field, di.Value, di.Reason)
		case errors.As(err, &te):
			s.logger.Error("create image: repo timeout", "err", err)
			return domain.PropertyImage{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("create image: repo create failed", "err", err)
			return domain.PropertyImage{}, apperrors.NewErrInternal("failed to create image")
		}
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
	if len(req.Files) > 110 {
		return nil, apperrors.NewErrInvalidInput("files", len(req.Files), "maximum 110 images allowed")
	}

	unlock := s.lockForProperty(req.PropertyID)
	defer unlock()

	// remove existing DB records for this property to ensure we replace them
	if _, err := s.repo.DeleteMany(ctx, req.PropertyID); err != nil {
		var nf dberrors.ErrNotFound
		if !errors.As(err, &nf) {
			s.logger.Error("create many: delete existing db rows failed", "err", err)
			return nil, apperrors.NewErrInternal("failed to remove existing images")
		}
	}
	// remove existing files on disk as well
	if delErr := s.store.DeletePropertyDir(req.PropertyID); delErr != nil {
		s.logger.Error("create many: filestore delete dir failed", "err", delErr, "property_id", req.PropertyID)
		return nil, apperrors.NewErrInternal("failed to remove existing images")
	}

	tmpDir, err := os.MkdirTemp(s.baseDir, fmt.Sprintf("%d_tmp_", req.PropertyID))
	if err != nil {
		s.logger.Error("create many: tmp dir failed", "err", err)
		return nil, apperrors.NewErrInternal("failed to prepare storage")
	}

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	for i, f := range req.Files {
		if f.Filename == "" || len(f.Data) == 0 {
			return nil, apperrors.NewErrInvalidInput("file", nil, "must be provided")
		}
		ext := filepath.Ext(f.Filename)
		fname := fmt.Sprintf("%d%s", i+1, ext)

		if _, err := s.store.SaveToDir(tmpDir, fname, f.Data); err != nil {
			return nil, s.mapFilestoreSaveError(err)
		}
	}

	finalDir := filepath.Join(s.baseDir, fmt.Sprintf("%d", req.PropertyID))

	if err := s.store.DeletePropertyDir(req.PropertyID); err != nil {
		s.logger.Error("create many: remove final dir failed", "err", err)
		return nil, apperrors.NewErrInternal("failed to prepare storage")
	}

	if err := os.Rename(tmpDir, finalDir); err != nil {
		s.logger.Error("create many: move temp to final failed", "err", err)
		return nil, apperrors.NewErrInternal("failed to store image files")
	}

	imgs := make([]domain.PropertyImage, 0, len(req.Files))
	for i := range req.Files {
		fname := fmt.Sprintf("%d%s", i+1, filepath.Ext(req.Files[i].Filename))

		if filepath.Ext(fname) == "" {

			entries, _ := os.ReadDir(finalDir)
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if strings.HasPrefix(e.Name(), fmt.Sprintf("%d.", i+1)) {
					fname = e.Name()
					break
				}
			}
		}
		imgs = append(imgs, domain.PropertyImage{PropertyID: req.PropertyID, Path: filepath.Join(finalDir, fname)})
	}

	ids, err := s.repo.CreateMany(ctx, imgs)
	if err != nil {
		var fk dberrors.ErrForeignKeyViolation
		var di dberrors.ErrInvalidInput
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &fk):

			_ = s.store.DeletePropertyDir(req.PropertyID)
			return nil, apperrors.NewErrNotFound("property", req.PropertyID)
		case errors.As(err, &di):
			_ = s.store.DeletePropertyDir(req.PropertyID)
			return nil, apperrors.NewErrInvalidInput(di.Field, di.Value, di.Reason)
		case errors.As(err, &te):
			s.logger.Error("create many images: repo timeout", "err", err)
			_ = s.store.DeletePropertyDir(req.PropertyID)
			return nil, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("create many images: repo failed", "err", err)

			if remErr := s.store.DeletePropertyDir(req.PropertyID); remErr != nil {
				s.logger.Error("create many: cleanup final dir failed", "err", remErr)
			}
			return nil, apperrors.NewErrInternal("failed to create images")
		}
	}

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
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return domain.PropertyImage{}, apperrors.NewErrNotFound("property_image", id)
		case errors.As(err, &te):
			s.logger.Error("get image: repo timeout", "err", err)
			return domain.PropertyImage{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("get image: repo error", "err", err)
			return domain.PropertyImage{}, apperrors.NewErrInternal("failed to fetch image")
		}
	}
	return img, nil
}

func (s *service) mapFilestoreSaveError(err error) error {
	var fi filestore.ErrInvalidInput
	var us filestore.ErrUnsupportedFormat
	var st filestore.ErrStorage
	switch {
	case errors.As(err, &fi):
		return apperrors.NewErrInvalidInput(fi.Field, fi.Value, fi.Reason)
	case errors.As(err, &us):
		return apperrors.NewErrInvalidInput("file", us.Filename, us.Detected)
	case errors.As(err, &st):
		s.logger.Error("filestore storage error", "err", err)
		return apperrors.NewErrInternal("failed to save image")
	default:
		s.logger.Error("filestore unknown error", "err", err)
		return apperrors.NewErrInternal("failed to save image")
	}
}

func (s *service) ListByProperty(ctx context.Context, propertyID int) ([]dto.ImageDTO, error) {

	unlock := s.lockForProperty(propertyID)
	defer unlock()

	list, err := s.repo.ListByProperty(ctx, propertyID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var di dberrors.ErrInvalidInput
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return nil, apperrors.NewErrNotFound("property", propertyID)
		case errors.As(err, &di):
			return nil, apperrors.NewErrInvalidInput(di.Field, di.Value, di.Reason)
		case errors.As(err, &te):
			s.logger.Error("list images: repo timeout", "err", err)
			return nil, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("list images: repo failed", "err", err)
			return nil, apperrors.NewErrInternal("failed to list images")
		}
	}

	files := make([]dto.ImageFile, 0, len(list))
	for _, img := range list {
		data, err := s.store.Read(img.Path)
		if err != nil {

			var fi filestore.ErrInvalidInput
			var st filestore.ErrStorage
			switch {
			case errors.As(err, &fi):
				return nil, apperrors.NewErrInvalidInput(fi.Field, fi.Value, fi.Reason)
			case errors.As(err, &st):

				if st.Operation == "read" && strings.Contains(strings.ToLower(st.Details), "not found") {
					return nil, apperrors.NewErrNotFound("property_image_file", img.ID)
				}
				s.logger.Error("list images: filestore storage error", "err", err, "path", img.Path)
				return nil, apperrors.NewErrInternal("failed to read image files")
			default:
				s.logger.Error("list images: filestore read unknown error", "err", err, "path", img.Path)
				return nil, apperrors.NewErrInternal("failed to read image files")
			}
		}
		fname := filepath.Base(img.Path)
		files = append(files, dto.ImageFile{Filename: fname, Data: data})
	}

	if len(files) == 0 {
		return []dto.ImageDTO{}, nil
	}
	return []dto.ImageDTO{{PropertyID: propertyID, Files: files}}, nil
}

func (s *service) Delete(ctx context.Context, id int) (int, error) {
	deletedID, err := s.repo.Delete(ctx, id)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return 0, apperrors.NewErrNotFound("property_image", id)
		case errors.As(err, &te):
			s.logger.Error("delete image: repo timeout", "err", err)
			return 0, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("delete image: repo failed", "err", err)
			return 0, apperrors.NewErrInternal("failed to delete image")
		}
	}
	return deletedID, nil
}
