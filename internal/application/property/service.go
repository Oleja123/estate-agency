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
	geocoder "github.com/Oleja123/estate-agency/internal/infrastructure/geocoder"
)

var _ Service = (*service)(nil)

type service struct {
	repo     domain.Repository
	typeRepo ptypedomain.Repository
	logger   *slog.Logger
	geo      geocoder.GeoService
	favSvc   favoritesvc.Service
}

func New(repo domain.Repository, typeRepo ptypedomain.Repository, logger *slog.Logger, geo geocoder.GeoService, favSvc favoritesvc.Service) Service {
	if geo == nil {
		geo = geocoder.NewNoop()
	}
	return &service{repo: repo, typeRepo: typeRepo, logger: logger, geo: geo, favSvc: favSvc}
}

func (s *service) Create(ctx context.Context, userID int, req dto.CreatePropertyRequest) (domain.Property, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.Property{}, apperrors.NewErrInvalidInput("title", title, "must not be empty")
	}

	if req.TypeID == 0 {
		return domain.Property{}, apperrors.NewErrInvalidInput("type_id", req.TypeID, "must be provided")
	}
	if _, err := s.typeRepo.GetByID(ctx, req.TypeID); err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return domain.Property{}, apperrors.NewErrNotFound("property_type", req.TypeID)
		default:
			s.logger.Error("create property: type repo error", "err", err)
			return domain.Property{}, apperrors.NewErrInternal("failed to validate property type")
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
		return domain.Property{}, apperrors.NewErrInvalidInput("property_address", p.PropertyAddress, "must not be empty")
	}

	lat, lon, err := s.geo.Geocode(p.PropertyAddress)
	if err != nil {
		s.logger.Error("geocode error при создании свойства", "err", err)
		// try to extract infra geocoder error details and translate to structured app error
		var (
			greq geocoder.ErrGeoRequest
			gno  geocoder.ErrGeoNoResults
			gcfg geocoder.ErrGeoConfig
			gdec geocoder.ErrGeoDecode
		)
		switch {
		case errors.As(err, &gno):
			appErr := apperrors.NewErrGeocodingWithDetails(gno.Error(), "geoapify", "no_results", "no coordinates for address", 422, gno.Address)
			return domain.Property{}, appErr
		case errors.As(err, &gcfg):
			appErr := apperrors.NewErrGeocodingWithDetails(gcfg.Error(), "geoapify", "config", gcfg.Message, 500, "")
			return domain.Property{}, appErr
		case errors.As(err, &gdec):
			appErr := apperrors.NewErrGeocodingWithDetails(gdec.Error(), "geoapify", "decode", gdec.Details, 502, "")
			return domain.Property{}, appErr
		case errors.As(err, &greq):
			appErr := apperrors.NewErrGeocodingWithDetails(greq.Error(), "geoapify", "request", greq.Details, 502, "")
			return domain.Property{}, appErr
		default:
			return domain.Property{}, apperrors.NewErrGeocoding(err.Error())
		}
	}
	p.Latitude = lat
	p.Longitude = lon

	id, err := s.repo.Create(ctx, p)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		switch {
		case errors.As(err, &ae):
			return domain.Property{}, apperrors.NewErrAlreadyExists("property", "title", title)
		default:
			s.logger.Error("create property: repo create error", "err", err)
			return domain.Property{}, apperrors.NewErrInternal("failed to create property")
		}
	}

	created, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("create property: failed to fetch created", "err", err)
		return domain.Property{}, apperrors.NewErrInternal("failed to fetch created property")
	}
	s.logger.Info("property created", "id", created.ID, "title", created.Title)
	return created, nil
}

func (s *service) GetByID(ctx context.Context, id int) (domain.Property, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return domain.Property{}, apperrors.NewErrNotFound("property", id)
		default:
			s.logger.Error("get property: repo error", "err", err)
			return domain.Property{}, apperrors.NewErrInternal("failed to fetch property")
		}
	}
	return p, nil
}

func (s *service) GetByIDWithFavorite(ctx context.Context, id int, userID int) (domain.Property, bool, error) {
	p, fav, err := s.repo.GetByIDWithFavorite(ctx, id, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return domain.Property{}, false, apperrors.NewErrNotFound("property", id)
		default:
			s.logger.Error("get property with favorite: repo error", "err", err)
			return domain.Property{}, false, apperrors.NewErrInternal("failed to fetch property")
		}
	}
	return p, fav, nil
}

func (s *service) Update(ctx context.Context, req dto.UpdatePropertyRequest) (domain.Property, error) {
	p, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		var nf dberrors.ErrNotFound
		switch {
		case errors.As(err, &nf):
			return domain.Property{}, apperrors.NewErrNotFound("property", req.ID)
		default:
			s.logger.Error("update property: failed to fetch", "err", err)
			return domain.Property{}, apperrors.NewErrInternal("failed to fetch property")
		}
	}

	if req.TypeID.Defined && req.TypeID.Valid {
		val := *req.TypeID.Value
		if val != 0 && val != p.TypeID {
			if _, err := s.typeRepo.GetByID(ctx, val); err != nil {
				var nf dberrors.ErrNotFound
				switch {
				case errors.As(err, &nf):
					return domain.Property{}, apperrors.NewErrNotFound("property_type", val)
				default:
					s.logger.Error("update property: type repo error", "err", err)
					return domain.Property{}, apperrors.NewErrInternal("failed to validate property type")
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
			return domain.Property{}, apperrors.NewErrInvalidInput("property_address", p.PropertyAddress, "must not be empty")
		}
		lat, lon, err := s.geo.Geocode(p.PropertyAddress)
		if err != nil {
			s.logger.Error("geocode error при обновлении свойства", "err", err)
			var (
				greq geocoder.ErrGeoRequest
				gno  geocoder.ErrGeoNoResults
				gcfg geocoder.ErrGeoConfig
				gdec geocoder.ErrGeoDecode
			)
			switch {
			case errors.As(err, &gno):
				appErr := apperrors.NewErrGeocodingWithDetails(gno.Error(), "geoapify", "no_results", "no coordinates for address", 422, gno.Address)
				return domain.Property{}, appErr
			case errors.As(err, &gcfg):
				appErr := apperrors.NewErrGeocodingWithDetails(gcfg.Error(), "geoapify", "config", gcfg.Message, 500, "")
				return domain.Property{}, appErr
			case errors.As(err, &gdec):
				appErr := apperrors.NewErrGeocodingWithDetails(gdec.Error(), "geoapify", "decode", gdec.Details, 502, "")
				return domain.Property{}, appErr
			case errors.As(err, &greq):
				appErr := apperrors.NewErrGeocodingWithDetails(greq.Error(), "geoapify", "request", greq.Details, 502, "")
				return domain.Property{}, appErr
			default:
				return domain.Property{}, apperrors.NewErrGeocoding(err.Error())
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
			return domain.Property{}, apperrors.NewErrAlreadyExists("property", ae.Field, ae.Value)
		case errors.As(err, &di):
			return domain.Property{}, apperrors.NewErrInvalidInput(di.Field, di.Value, di.Reason)
		default:
			s.logger.Error("update property: repo update failed", "err", err)
			return domain.Property{}, apperrors.NewErrInternal("failed to update property")
		}
	}
	s.logger.Info("update property: updated", "id", updated.ID)
	return updated, nil
}

func (s *service) List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error) {
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	list, total, err := s.repo.List(ctx, dr)
	if err != nil {
		s.logger.Error("list properties: repo list failed", "err", err)
		return dto.ListPropertiesResponse{}, apperrors.NewErrInternal("failed to list properties")
	}
	props := MapProperties(list)
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
			s.logger.Error("delete property: repo delete failed", "err", err)
			return 0, apperrors.NewErrInternal("failed to delete property")
		}
	}
	s.logger.Info("delete property: deleted", "id", deletedID)
	return deletedID, nil
}

func (s *service) ToggleFavorite(ctx context.Context, userID int, propertyID int) (bool, domain.Property, error) {
	if s.favSvc == nil {
		return false, domain.Property{}, apperrors.NewErrInternal("favorites not configured")
	}
	key := favdto.CreateFavoriteRequest{UserID: userID, PropertyID: propertyID}
	exists, err := s.favSvc.Exists(ctx, key)
	if err != nil {
		return false, domain.Property{}, err
	}
	if exists {
		_, err := s.favSvc.Delete(ctx, key)
		if err != nil {
			return false, domain.Property{}, err
		}
		return false, domain.Property{}, nil
	}
	// create favorite returns the property id; fetch full property to return to caller
	_, err = s.favSvc.Create(ctx, key)
	if err != nil {
		return false, domain.Property{}, err
	}
	createdProp, err := s.favSvc.GetByUserAndProperty(ctx, key)
	if err != nil {
		return false, domain.Property{}, err
	}
	return true, createdProp, nil
}
