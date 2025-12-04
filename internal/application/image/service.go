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
		return domain.PropertyImage{}, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "обязательное поле")
	}
	if req.File.Filename == "" || len(req.File.Data) == 0 {
		return domain.PropertyImage{}, apperrors.NewErrInvalidInput("file", nil, "обязательное поле")
	}

	unlock := s.lockForProperty(req.PropertyID)
	defer unlock()

	existing, err := s.repo.ListByProperty(ctx, req.PropertyID)
	if err != nil {
		s.logger.Error("создание изображения: не удалось получить список существующих", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось подготовить изображения")
	}
	if len(existing) > 0 {
		if _, err := s.repo.DeleteMany(ctx, req.PropertyID); err != nil {

			var nf dberrors.ErrNotFound
			if !errors.As(err, &nf) {
				s.logger.Error("создание изображения: не удалось удалить существующие строки в базе данных", "err", err)
				return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось удалить существующие изображения")
			}
		}
		if delErr := s.store.DeletePropertyDir(req.PropertyID); delErr != nil {
			s.logger.Error("создание изображения: ошибка удаления директории в filestore", "err", delErr, "property_id", req.PropertyID)
			return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось удалить существующие изображения")
		}
	}

	tmpDir, err := os.MkdirTemp(s.baseDir, fmt.Sprintf("%d_tmp_", req.PropertyID))
	if err != nil {
		s.logger.Error("создание изображения: ошибка временной директории", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось подготовить хранилище")
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
		s.logger.Error("создание изображения: не удалось удалить финальную директорию", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось подготовить хранилище")
	}

	if err := os.Rename(tmpDir, finalDir); err != nil {
		s.logger.Error("создание изображения: не удалось переместить временные файлы в финальную директорию", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось сохранить файлы изображений")
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
			s.logger.Error("создание изображения: превышено время ожидания репозитория", "err", err)
			return domain.PropertyImage{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("создание изображения: ошибка создания в репозитории", "err", err)
			return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось создать изображение")
		}
	}

	created, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("создание изображения: не удалось получить созданное изображение", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось получить созданное изображение")
	}
	return created, nil
}

func (s *service) CreateMany(ctx context.Context, req dto.CreateImagesRequest) ([]domain.PropertyImage, error) {
	if req.PropertyID == 0 {
		return nil, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "обязательное поле")
	}
	if len(req.Files) > 110 {
		return nil, apperrors.NewErrInvalidInput("files", len(req.Files), "максимум 110 изображений")
	}

	unlock := s.lockForProperty(req.PropertyID)
	defer unlock()

	if _, err := s.repo.DeleteMany(ctx, req.PropertyID); err != nil {
		var nf dberrors.ErrNotFound
		if !errors.As(err, &nf) {
			s.logger.Error("создание изображений: не удалось удалить существующие строки в базе данных", "err", err)
			return nil, apperrors.NewErrInternal("не удалось удалить существующие изображения")
		}
	}

	if delErr := s.store.DeletePropertyDir(req.PropertyID); delErr != nil {
		s.logger.Error("создание изображений: ошибка удаления директории в filestore", "err", delErr, "property_id", req.PropertyID)
		return nil, apperrors.NewErrInternal("не удалось удалить существующие изображения")
	}

	tmpDir, err := os.MkdirTemp(s.baseDir, fmt.Sprintf("%d_tmp_", req.PropertyID))
	if err != nil {
		s.logger.Error("create many: tmp dir failed", "err", err)
		return nil, apperrors.NewErrInternal("не удалось подготовить хранилище")
	}

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	for i, f := range req.Files {
		if f.Filename == "" || len(f.Data) == 0 {
			return nil, apperrors.NewErrInvalidInput("file", nil, "обязательное поле")
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
		return nil, apperrors.NewErrInternal("не удалось подготовить хранилище")
	}

	if err := os.Rename(tmpDir, finalDir); err != nil {
		s.logger.Error("create many: move temp to final failed", "err", err)
		return nil, apperrors.NewErrInternal("не удалось сохранить файлы изображений")
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
			return nil, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("create many images: repo failed", "err", err)

			if remErr := s.store.DeletePropertyDir(req.PropertyID); remErr != nil {
				s.logger.Error("create many: cleanup final dir failed", "err", remErr)
			}
			return nil, apperrors.NewErrInternal("не удалось создать изображения")
		}
	}

	var created []domain.PropertyImage
	for _, id := range ids {
		img, err := s.repo.GetByID(ctx, id)
		if err != nil {
			s.logger.Error("создание нескольких изображений: не удалось получить созданное изображение", "err", err, "id", id)
			return nil, apperrors.NewErrInternal("не удалось получить созданные изображения")
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
			return domain.PropertyImage{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("get image: repo error", "err", err)
			return domain.PropertyImage{}, apperrors.NewErrInternal("не удалось получить изображение")
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
		return apperrors.NewErrInternal("не удалось сохранить изображение")
	default:
		s.logger.Error("filestore unknown error", "err", err)
		return apperrors.NewErrInternal("не удалось сохранить изображение")
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
			return nil, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("list images: repo failed", "err", err)
			return nil, apperrors.NewErrInternal("не удалось получить список изображений")
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
				return nil, apperrors.NewErrInternal("не удалось прочитать файлы изображений")
			default:
				s.logger.Error("list images: filestore read unknown error", "err", err, "path", img.Path)
				return nil, apperrors.NewErrInternal("не удалось прочитать файлы изображений")
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
			return 0, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("delete image: repo failed", "err", err)
			return 0, apperrors.NewErrInternal("не удалось удалить изображение")
		}
	}
	return deletedID, nil
}
