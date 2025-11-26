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
	locks   sync.Map // map[int]*sync.Mutex
}

// lockForProperty returns an unlock function that locks per-property mutex.
// It uses sync.Map to store *sync.Mutex instances keyed by propertyID.
func (s *service) lockForProperty(propertyID int) func() {
	if propertyID == 0 {
		// nothing to lock for invalid id
		return func() {}
	}
	v, _ := s.locks.LoadOrStore(propertyID, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return func() {
		m.Unlock()
	}
}

// New constructs the image service. basePath is the root directory where images will be saved
// (e.g. "property_images"). Use an absolute or relative path. Tests may pass t.TempDir().
func New(repo domain.ImageRepository, logger *slog.Logger, basePath string) Service {
	if basePath == "" {
		basePath = "property_images"
	}
	store := filestore.New(basePath)
	return &service{repo: repo, logger: logger, store: store, baseDir: basePath}
}

func (s *service) Create(ctx context.Context, req dto.CreateImageRequest) (domain.PropertyImage, error) {
	if req.PropertyID == 0 {
		return domain.PropertyImage{}, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "must be provided")
	}
	if req.File.Filename == "" || len(req.File.Data) == 0 {
		return domain.PropertyImage{}, apperrors.NewErrInvalidInput("file", nil, "must be provided")
	}

	// per-property lock to avoid concurrent mutations for the same property
	unlock := s.lockForProperty(req.PropertyID)
	defer unlock()

	// remove existing images for this property in bulk (DB rows + files directory)
	existing, err := s.repo.ListByProperty(ctx, req.PropertyID)
	if err != nil {
		s.logger.Error("create image: list existing failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to prepare images")
	}
	if len(existing) > 0 {
		if err := s.repo.DeleteMany(ctx, req.PropertyID); err != nil {
			// If the rows are already gone it's fine; otherwise treat as internal
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

	// Atomic-ish flow: write to temp dir, move into final dir, then create DB row.
	tmpDir, err := os.MkdirTemp(s.baseDir, fmt.Sprintf("%d_tmp_", req.PropertyID))
	if err != nil {
		s.logger.Error("create image: tmp dir failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to prepare storage")
	}
	// ensure cleanup on failure
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Save into tmp dir. If filename has no ext, filestore will append detected ext.
	fname := fmt.Sprintf("%d%s", 1, filepath.Ext(req.File.Filename))
	// if ext empty, SaveToDir will append detected one
	fullTmpPath, err := s.store.SaveToDir(tmpDir, fname, req.File.Data)
	if err != nil {
		return domain.PropertyImage{}, s.mapFilestoreSaveError(err)
	}

	finalDir := filepath.Join(s.baseDir, fmt.Sprintf("%d", req.PropertyID))
	// remove existing final dir (best-effort)
	if err := s.store.DeletePropertyDir(req.PropertyID); err != nil {
		s.logger.Error("create image: remove final dir failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to prepare storage")
	}

	// move tmp -> final
	if err := os.Rename(tmpDir, finalDir); err != nil {
		s.logger.Error("create image: move temp to final failed", "err", err)
		return domain.PropertyImage{}, apperrors.NewErrInternal("failed to store image files")
	}

	finalPath := filepath.Join(finalDir, filepath.Base(fullTmpPath))

	img := domain.PropertyImage{PropertyID: req.PropertyID, Path: finalPath}
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
	if len(req.Files) > 110 {
		return nil, apperrors.NewErrInvalidInput("files", len(req.Files), "maximum 110 images allowed")
	}

	// Strategy: write new files into a temp dir, atomically move them to final dir,
	// then insert DB rows. On DB failure we attempt to remove moved files to avoid orphans.

	// per-property lock to avoid concurrent uploads for the same property
	unlock := s.lockForProperty(req.PropertyID)
	defer unlock()

	tmpDir, err := os.MkdirTemp(s.baseDir, fmt.Sprintf("%d_tmp_", req.PropertyID))
	if err != nil {
		s.logger.Error("create many: tmp dir failed", "err", err)
		return nil, apperrors.NewErrInternal("failed to prepare storage")
	}
	// cleanup tmp on exit (if still exists)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Save each file into tmpDir
	for i, f := range req.Files {
		if f.Filename == "" || len(f.Data) == 0 {
			return nil, apperrors.NewErrInvalidInput("file", nil, "must be provided")
		}
		ext := filepath.Ext(f.Filename)
		fname := fmt.Sprintf("%d%s", i+1, ext)
		// SaveToDir will append detected extension if ext is empty
		if _, err := s.store.SaveToDir(tmpDir, fname, f.Data); err != nil {
			return nil, s.mapFilestoreSaveError(err)
		}
	}

	finalDir := filepath.Join(s.baseDir, fmt.Sprintf("%d", req.PropertyID))
	// remove existing final dir (best-effort)
	if err := s.store.DeletePropertyDir(req.PropertyID); err != nil {
		s.logger.Error("create many: remove final dir failed", "err", err)
		return nil, apperrors.NewErrInternal("failed to prepare storage")
	}

	// move tmp -> final
	if err := os.Rename(tmpDir, finalDir); err != nil {
		s.logger.Error("create many: move temp to final failed", "err", err)
		return nil, apperrors.NewErrInternal("failed to store image files")
	}

	// Build images metadata pointing to final paths
	imgs := make([]domain.PropertyImage, 0, len(req.Files))
	for i := range req.Files {
		fname := fmt.Sprintf("%d%s", i+1, filepath.Ext(req.Files[i].Filename))
		// if original had no ext, we should detect the file to get ext
		if filepath.Ext(fname) == "" {
			// try to find file in finalDir by scanning created files — fallback to filename in tmp/ final
			// but simplest: use the file that exists with a matching prefix
			// attempt to detect ext by looking for files like "i+1.*"
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
		s.logger.Error("create many images: repo failed", "err", err)
		// attempt to cleanup final dir to avoid orphan files
		if remErr := s.store.DeletePropertyDir(req.PropertyID); remErr != nil {
			s.logger.Error("create many: cleanup final dir failed", "err", remErr)
		}
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

// mapFilestoreSaveError translates filestore errors into application errors and logs storage
// failures. It centralizes error handling for Save operations.
func (s *service) mapFilestoreSaveError(err error) error {
	var fi filestore.ErrInvalidInput
	var us filestore.ErrUnsupportedFormat
	var st filestore.ErrStorage
	if errors.As(err, &fi) {
		return apperrors.NewErrInvalidInput(fi.Field, fi.Value, fi.Reason)
	}
	if errors.As(err, &us) {
		return apperrors.NewErrInvalidInput("file", us.Filename, us.Detected)
	}
	if errors.As(err, &st) {
		s.logger.Error("filestore storage error", "err", err)
		return apperrors.NewErrInternal("failed to save image")
	}
	s.logger.Error("filestore unknown error", "err", err)
	return apperrors.NewErrInternal("failed to save image")
}

func (s *service) ListByProperty(ctx context.Context, propertyID int) ([]dto.ImageDTO, error) {
	// take per-property lock to ensure consistency between DB rows and filesystem during
	// concurrent mutations (Create/CreateMany).
	unlock := s.lockForProperty(propertyID)
	defer unlock()
	// fetch domain images and map to DTOs
	list, err := s.repo.ListByProperty(ctx, propertyID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var di dberrors.ErrInvalidInput
		if errors.As(err, &nf) {
			return nil, apperrors.NewErrNotFound("property", propertyID)
		}
		if errors.As(err, &di) {
			return nil, apperrors.NewErrInvalidInput(di.Field, di.Value, di.Reason)
		}
		s.logger.Error("list images: repo failed", "err", err)
		return nil, apperrors.NewErrInternal("failed to list images")
	}

	dtos := make([]dto.ImageDTO, 0, len(list))
	for _, img := range list {
		data, err := s.store.Read(img.Path)
		if err != nil {
			// Distinguish filestore errors: missing file -> NotFound, invalid input -> InvalidInput, storage -> Internal
			var fi filestore.ErrInvalidInput
			var st filestore.ErrStorage
			if errors.As(err, &fi) {
				return nil, apperrors.NewErrInvalidInput(fi.Field, fi.Value, fi.Reason)
			}
			if errors.As(err, &st) {
				// If read failed because file not found, return NotFound to client
				if st.Operation == "read" && strings.Contains(strings.ToLower(st.Details), "not found") {
					return nil, apperrors.NewErrNotFound("property_image_file", img.ID)
				}
				s.logger.Error("list images: filestore storage error", "err", err, "path", img.Path)
				return nil, apperrors.NewErrInternal("failed to read image files")
			}
			s.logger.Error("list images: filestore read unknown error", "err", err, "path", img.Path)
			return nil, apperrors.NewErrInternal("failed to read image files")
		}
		fname := filepath.Base(img.Path)
		dtos = append(dtos, dto.ImageDTO{ID: img.ID, PropertyID: img.PropertyID, Filename: fname, Data: data})
	}
	return dtos, nil
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
