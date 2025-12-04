package propertyservice

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favoritesvc "github.com/Oleja123/estate-agency/internal/application/favorite"
	favdto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
	ptypedomain "github.com/Oleja123/estate-agency/internal/domain/property_type"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/Oleja123/estate-agency/internal/infrastructure/filestore"
	geocoder "github.com/Oleja123/estate-agency/internal/infrastructure/geocoder"
)

var _ Service = (*service)(nil)

type service struct {
	repo      domain.Repository
	typeRepo  ptypedomain.Repository
	logger    *slog.Logger
	geo       geocoder.GeoService
	favSvc    favoritesvc.Service
	fileStore *filestore.FileStore
}

func New(repo domain.Repository, typeRepo ptypedomain.Repository, logger *slog.Logger, geo geocoder.GeoService, favSvc favoritesvc.Service, fileStore *filestore.FileStore) Service {
	if geo == nil {
		geo = geocoder.NewNoop()
	}
	return &service{repo: repo, typeRepo: typeRepo, logger: logger, geo: geo, favSvc: favSvc, fileStore: fileStore}
}

func (s *service) Create(ctx context.Context, userID int, req dto.CreatePropertyRequest) (dto.PropertyDTO, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return dto.PropertyDTO{}, apperrors.NewErrInvalidInput("title", title, "не может быть пустым")
	}

	if req.TypeID == 0 {
		return dto.PropertyDTO{}, apperrors.NewErrInvalidInput("type_id", req.TypeID, "обязательное поле")
	}
	if _, err := s.typeRepo.GetByID(ctx, req.TypeID); err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return dto.PropertyDTO{}, apperrors.NewErrNotFound("property_type", req.TypeID)
		default:
			s.logger.Error("создание свойства: ошибка репозитория типов", "err", err)
			return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось проверить тип свойства")
		}
	}

	p := domain.Property{
		Title:               title,
		PropertyDescription: strings.TrimSpace(req.PropertyDescription),
		TypeID:              req.TypeID,
		TransactionType:     domain.TransactionType(req.TransactionType),
		Price:               req.Price,
		Area:                req.Area,
		PropertyAddress:     strings.TrimSpace(req.PropertyAddress),
		City:                strings.TrimSpace(req.City),
		PropertyStatus:      domain.StatusActive,
		CreatedBy:           userID,
	}

	if p.PropertyAddress == "" {
		return dto.PropertyDTO{}, apperrors.NewErrInvalidInput("property_address", p.PropertyAddress, "не может быть пустым")
	}

	lat, lon, err := s.geo.Geocode(p.PropertyAddress)
	if err != nil {
		s.logger.Error("ошибка геокодирования при создании свойства", "err", err)

		var (
			greq geocoder.ErrGeoRequest
			gno  geocoder.ErrGeoNoResults
			gcfg geocoder.ErrGeoConfig
			gdec geocoder.ErrGeoDecode
		)
		switch {
		case errors.As(err, &gno):
			appErr := apperrors.NewErrGeocodingWithDetails(gno.Error(), "geoapify", "no_results", "нет координат для адреса", 422, gno.Address)
			return dto.PropertyDTO{}, appErr
		case errors.As(err, &gcfg):
			appErr := apperrors.NewErrGeocodingWithDetails(gcfg.Error(), "geoapify", "config", gcfg.Message, 500, "")
			return dto.PropertyDTO{}, appErr
		case errors.As(err, &gdec):
			appErr := apperrors.NewErrGeocodingWithDetails(gdec.Error(), "geoapify", "decode", gdec.Details, 502, "")
			return dto.PropertyDTO{}, appErr
		case errors.As(err, &greq):
			appErr := apperrors.NewErrGeocodingWithDetails(greq.Error(), "geoapify", "request", greq.Details, 502, "")
			return dto.PropertyDTO{}, appErr
		default:
			return dto.PropertyDTO{}, apperrors.NewErrGeocoding(err.Error())
		}
	}
	p.Latitude = lat
	p.Longitude = lon

	id, err := s.repo.Create(ctx, p)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		switch {
		case errors.As(err, &ae):
			return dto.PropertyDTO{}, apperrors.NewErrAlreadyExists("property", "title", title)
		default:
			s.logger.Error("создание свойства: ошибка при создании в репозитории", "err", err)
			return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось создать свойство")
		}
	}

	created, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("создание свойства: не удалось получить созданное свойство", "err", err)
		return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось получить созданное свойство")
	}
	s.logger.Info("объявление создано", "id", created.ID, "title", created.Title)
	return s.MapProperty(created)
}

func (s *service) GetByID(ctx context.Context, id int) (dto.PropertyDTO, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return dto.PropertyDTO{}, apperrors.NewErrNotFound("property", id)
		default:
			s.logger.Error("получение свойства: ошибка репозитория", "err", err)
			return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось получить свойство")
		}
	}
	return s.MapProperty(p)
}

func (s *service) GetByIDWithFavorite(ctx context.Context, id int, userID int) (dto.PropertyDTO, error) {
	p, fav, err := s.repo.GetByIDWithFavorite(ctx, id, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return dto.PropertyDTO{}, apperrors.NewErrNotFound("property", id)
		default:
			s.logger.Error("получение свойства с информацией об избранном: ошибка репозитория", "err", err)
			return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось получить свойство")
		}
	}
	return s.MapPropertyWithFavorite(p, fav)
}

func (s *service) Update(ctx context.Context, req dto.UpdatePropertyRequest) (dto.PropertyDTO, error) {
	p, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return dto.PropertyDTO{}, apperrors.NewErrNotFound("property", req.ID)
		default:
			s.logger.Error("обновление свойства: не удалось получить свойство", "err", err)
			return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось получить свойство")
		}
	}

	if req.TypeID.Defined && req.TypeID.Valid {
		val := *req.TypeID.Value
		if val != 0 && val != p.TypeID {
			if _, err := s.typeRepo.GetByID(ctx, val); err != nil {
				var nf dberrors.ErrNotFound
				switch {
				case errors.As(err, &nf):
					return dto.PropertyDTO{}, apperrors.NewErrNotFound("property_type", val)
				default:
					s.logger.Error("обновление свойства: ошибка репозитория типов", "err", err)
					return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось проверить тип свойства")
				}
			}
			p.TypeID = val
		}
	}

	if req.Title.Defined && req.Title.Valid {
		p.Title = strings.TrimSpace(*req.Title.Value)
	}
	if req.PropertyDescription.Defined && req.PropertyDescription.Valid {
		p.PropertyDescription = strings.TrimSpace(*req.PropertyDescription.Value)
	}
	if req.TransactionType.Defined && req.TransactionType.Valid {
		p.TransactionType = domain.TransactionType(*req.TransactionType.Value)
	}
	if req.Price.Defined && req.Price.Valid {
		p.Price = *req.Price.Value
	}
	if req.Area.Defined && req.Area.Valid {
		p.Area = *req.Area.Value
	}
	if req.PropertyAddress.Defined && req.PropertyAddress.Valid {
		p.PropertyAddress = strings.TrimSpace(*req.PropertyAddress.Value)
	}
	if req.City.Defined && req.City.Valid {
		p.City = strings.TrimSpace(*req.City.Value)
	}
	if req.PropertyStatus.Defined && req.PropertyStatus.Valid {
		p.PropertyStatus = domain.PropertyStatus(*req.PropertyStatus.Value)
	}

	if req.PropertyAddress.Defined && req.PropertyAddress.Valid {
		if p.PropertyAddress == "" {
			return dto.PropertyDTO{}, apperrors.NewErrInvalidInput("property_address", p.PropertyAddress, "не может быть пустым")
		}
		lat, lon, err := s.geo.Geocode(p.PropertyAddress)
		if err != nil {
			s.logger.Error("ошибка геокодирования при обновлении свойства", "err", err)
			var (
				greq geocoder.ErrGeoRequest
				gno  geocoder.ErrGeoNoResults
				gcfg geocoder.ErrGeoConfig
				gdec geocoder.ErrGeoDecode
			)
			switch {
			case errors.As(err, &gno):
				appErr := apperrors.NewErrGeocodingWithDetails(gno.Error(), "geoapify", "no_results", "нет координат для адреса", 422, gno.Address)
				return dto.PropertyDTO{}, appErr
			case errors.As(err, &gcfg):
				appErr := apperrors.NewErrGeocodingWithDetails(gcfg.Error(), "geoapify", "config", gcfg.Message, 500, "")
				return dto.PropertyDTO{}, appErr
			case errors.As(err, &gdec):
				appErr := apperrors.NewErrGeocodingWithDetails(gdec.Error(), "geoapify", "decode", gdec.Details, 502, "")
				return dto.PropertyDTO{}, appErr
			case errors.As(err, &greq):
				appErr := apperrors.NewErrGeocodingWithDetails(greq.Error(), "geoapify", "request", greq.Details, 502, "")
				return dto.PropertyDTO{}, appErr
			default:
				return dto.PropertyDTO{}, apperrors.NewErrGeocoding(err.Error())
			}
		}
		p.Latitude = lat
		p.Longitude = lon
	}

	updated, err := s.repo.Update(ctx, p)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		var di dberrors.ErrInvalidInput
		switch {
		case errors.As(err, &ae):
			return dto.PropertyDTO{}, apperrors.NewErrAlreadyExists("property", ae.Field, ae.Value)
		case errors.As(err, &di):
			return dto.PropertyDTO{}, apperrors.NewErrInvalidInput(di.Field, di.Value, di.Reason)
		default:
			s.logger.Error("обновление свойства: ошибка обновления в репозитории", "err", err)
			return dto.PropertyDTO{}, apperrors.NewErrInternal("не удалось обновить свойство")
		}
	}
	s.logger.Info("обновление объявления: обновлено", "id", updated.ID)
	return s.MapProperty(updated)
}

func (s *service) List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error) {
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	list, total, err := s.repo.List(ctx, dr)
	if err != nil {
		s.logger.Error("получение списка свойств: ошибка репозитория", "err", err)
		return dto.ListPropertiesResponse{}, apperrors.NewErrInternal("не удалось получить список свойств")
	}
	props, err := s.MapProperties(list)
	if err != nil {
		return dto.ListPropertiesResponse{}, err
	}
	return dto.ListPropertiesResponse{Properties: props, Total: total}, nil
}

func (s *service) Delete(ctx context.Context, id int) (int, error) {
	deletedID, err := s.repo.Delete(ctx, id)
	if err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return 0, apperrors.NewErrNotFound("property", id)
		default:
			s.logger.Error("удаление свойства: ошибка удаления в репозитории", "err", err)
			return 0, apperrors.NewErrInternal("не удалось удалить свойство")
		}
	}
	s.logger.Info("удаление объявления: удалено", "id", deletedID)
	return deletedID, nil
}

func (s *service) ToggleFavorite(ctx context.Context, userID int, propertyID int) (bool, dto.PropertyDTO, error) {
	if s.favSvc == nil {
		return false, dto.PropertyDTO{}, apperrors.NewErrInternal("сервис избранного не настроен")
	}
	key := favdto.CreateFavoriteRequest{UserID: userID, PropertyID: propertyID}
	exists, err := s.favSvc.Exists(ctx, key)
	if err != nil {
		return false, dto.PropertyDTO{}, err
	}
	if exists {
		_, err := s.favSvc.Delete(ctx, key)
		if err != nil {
			return false, dto.PropertyDTO{}, err
		}
		return false, dto.PropertyDTO{}, nil
	}

	_, err = s.favSvc.Create(ctx, key)
	if err != nil {
		return false, dto.PropertyDTO{}, err
	}
	createdProp, err := s.favSvc.GetByUserAndProperty(ctx, key)
	if err != nil {
		return false, dto.PropertyDTO{}, err
	}
	dtoProp, err := s.MapProperty(createdProp)
	if err != nil {
		return false, dto.PropertyDTO{}, err
	}
	return true, dtoProp, nil
}
