package favoriteservice

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favdto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	imagedto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	propertydto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	imagedomain "github.com/Oleja123/estate-agency/internal/domain/image"
	prop "github.com/Oleja123/estate-agency/internal/domain/property"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/Oleja123/estate-agency/internal/infrastructure/filestore"
)

var _ Service = (*service)(nil)

type service struct {
	repo      domain.Repository
	logger    *slog.Logger
	fileStore *filestore.FileStore
}

func New(repo domain.Repository, logger *slog.Logger, fileStore *filestore.FileStore) Service {
	return &service{repo: repo, logger: logger, fileStore: fileStore}
}

func (s *service) Create(ctx context.Context, req favdto.CreateFavoriteRequest) (int, error) {
	if req.UserID == 0 {
		return 0, apperrors.NewErrInvalidInput("user_id", req.UserID, "обязательное поле")
	}
	if req.PropertyID == 0 {
		return 0, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "обязательное поле")
	}

	fav := domain.Favorite{UserID: req.UserID, PropertyID: req.PropertyID}
	if err := s.repo.Create(ctx, fav); err != nil {
		var ae dberrors.ErrAlreadyExists
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &ae):
			return 0, apperrors.NewErrAlreadyExists("favorite", "user_id,property_id", nil)
		case errors.As(err, &te):
			s.logger.Error("создание избранного: превышено время ожидания репозитория", "err", err)
			return 0, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("создание избранного: ошибка репозитория", "err", err)
			return 0, apperrors.NewErrInternal("не удалось добавить в избранное")
		}
	}
	return req.PropertyID, nil
}

func (s *service) GetByUserAndProperty(ctx context.Context, key favdto.CreateFavoriteRequest) (prop.Property, error) {
	p, err := s.repo.GetByUserAndProperty(ctx, key.UserID, key.PropertyID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return prop.Property{}, apperrors.NewErrNotFound("favorite", map[string]int{"user_id": key.UserID, "property_id": key.PropertyID})
		case errors.As(err, &te):
			s.logger.Error("получение избранного: превышено время ожидания репозитория", "err", err)
			return prop.Property{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("получение избранного: ошибка репозитория", "err", err)
			return prop.Property{}, apperrors.NewErrInternal("не удалось получить избранное")
		}
	}
	return p, nil
}

func (s *service) Delete(ctx context.Context, key favdto.CreateFavoriteRequest) (int, error) {
	deletedID, err := s.repo.Delete(ctx, key.UserID, key.PropertyID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return 0, apperrors.NewErrNotFound("favorite", map[string]int{"user_id": key.UserID, "property_id": key.PropertyID})
		case errors.As(err, &te):
			s.logger.Error("удаление избранного: превышено время ожидания репозитория", "err", err)
			return 0, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("удаление избранного: ошибка репозитория", "err", err)
			return 0, apperrors.NewErrInternal("не удалось удалить избранное")
		}
	}
	return deletedID, nil
}

func (s *service) List(ctx context.Context, req favdto.ListFavoritesRequest) (favdto.ListFavoritesResponse, error) {
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	list, total, err := s.repo.List(ctx, dr)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("получение списка избранного: превышено время ожидания репозитория", "err", err)
			return favdto.ListFavoritesResponse{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("получение списка избранного: ошибка репозитория", "err", err)
			return favdto.ListFavoritesResponse{}, apperrors.NewErrInternal("не удалось получить список избранного")
		}
	}

	mapped, err := s.mapProperties(list)
	if err != nil {
		s.logger.Error("получение списка избранного: ошибка маппинга свойств", "err", err)
		return favdto.ListFavoritesResponse{}, apperrors.NewErrInternal("не удалось обработать список избранного")
	}
	return favdto.ListFavoritesResponse{Favorites: mapped, Total: total}, nil
}

func (s *service) mapProperties(list []prop.Property) ([]propertydto.PropertyDTO, error) {
	out := make([]propertydto.PropertyDTO, 0, len(list))
	for _, p := range list {
		mp, err := s.mapProperty(p)
		if err != nil {
			return nil, err
		}
		out = append(out, mp)
	}
	return out, nil
}

func (s *service) mapProperty(p prop.Property) (propertydto.PropertyDTO, error) {
	images, err := s.getFilesImages(p.Images)
	if err != nil {
		s.logger.Error("не удалось получить файлы изображений", "error", err)
		return propertydto.PropertyDTO{}, err
	}

	var image *imagedto.ImageFile
	if len(images) > 0 {
		image = &images[0]
	}

	return propertydto.PropertyDTO{
		ID:                  p.ID,
		Title:               p.Title,
		PropertyDescription: p.PropertyDescription,
		TypeID:              p.TypeID,
		TransactionType:     string(p.TransactionType),
		Price:               p.Price,
		Area:                p.Area,
		PropertyAddress:     p.PropertyAddress,
		Latitude:            p.Latitude,
		Longitude:           p.Longitude,
		City:                p.City,
		PropertyStatus:      string(p.PropertyStatus),
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
		Images:              images,
		Image:               image,
	}, nil
}

func (s *service) getFilesImages(images []imagedomain.PropertyImage) ([]imagedto.ImageFile, error) {
	var result []imagedto.ImageFile
	for _, img := range images {
		var data []byte
		if s.fileStore != nil {
			d, err := s.fileStore.Read(img.Path)
			if err != nil {
				s.logger.Error("не удалось прочитать файл изображения", "path", img.Path, "error", err)
				return nil, apperrors.NewErrInternal("не удалось открыть файл изображения")
			}
			data = d
		}
		file := imagedto.ImageFile{
			Filename: filepath.Base(img.Path),
			Data:     data,
		}
		result = append(result, file)
	}
	return result, nil
}

func (s *service) Exists(ctx context.Context, key favdto.CreateFavoriteRequest) (bool, error) {
	ok, err := s.repo.Exists(ctx, key.UserID, key.PropertyID)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("проверка избранного: превышено время ожидания репозитория", "err", err)
			return false, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("проверка избранного: ошибка репозитория", "err", err)
			return false, apperrors.NewErrInternal("не удалось проверить наличие в избранном")
		}
	}
	return ok, nil
}
