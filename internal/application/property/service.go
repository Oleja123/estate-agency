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
	favdomain "github.com/Oleja123/estate-agency/internal/domain/favorite"
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

// New constructs property service with property repository and property-type repository.
func New(repo domain.Repository, typeRepo ptypedomain.Repository, logger *slog.Logger, geo geocoder.GeoService, favSvc favoritesvc.Service) Service {
	if geo == nil {
		geo = geocoder.NewNoop()
	}
	return &service{repo: repo, typeRepo: typeRepo, logger: logger, geo: geo, favSvc: favSvc}
}

func (s *service) Create(ctx context.Context, req dto.CreatePropertyRequest) (domain.Property, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.Property{}, apperrors.NewErrInvalidInput("title", title, "must not be empty")
	}

	// validate type existence
	if req.TypeID == 0 {
		return domain.Property{}, apperrors.NewErrInvalidInput("type_id", req.TypeID, "must be provided")
	}
	if _, err := s.typeRepo.GetByID(ctx, req.TypeID); err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return domain.Property{}, apperrors.NewErrNotFound("property_type", req.TypeID)
		}
		s.logger.Error("create property: type repo error", "err", err)
		return domain.Property{}, apperrors.NewErrInternal("failed to validate property type")
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
		CreatedBy:           req.CreatedBy,
	}

	// attempt to geocode address (best-effort)
	if p.PropertyAddress != "" {
		if lat, lon, err := s.geo.Geocode(p.PropertyAddress); err == nil {
			p.Latitude = lat
			p.Longitude = lon
		} else {
			s.logger.DebugContext(context.Background(), "geocode failed", "err", err)
		}
	}

	id, err := s.repo.Create(ctx, p)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		if errors.As(err, &ae) {
			return domain.Property{}, apperrors.NewErrAlreadyExists("property", "title", title)
		}
		s.logger.Error("create property: repo create error", "err", err)
		return domain.Property{}, apperrors.NewErrInternal("failed to create property")
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
		if errors.As(err, &nf) {
			return domain.Property{}, apperrors.NewErrNotFound("property", id)
		}
		s.logger.Error("get property: repo error", "err", err)
		return domain.Property{}, apperrors.NewErrInternal("failed to fetch property")
	}
	return p, nil
}

func (s *service) Update(ctx context.Context, req dto.UpdatePropertyRequest) error {
	p, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("property", req.ID)
		}
		s.logger.Error("update property: failed to fetch", "err", err)
		return apperrors.NewErrInternal("failed to fetch property")
	}

	// if type changed (non-zero), validate existence
	if req.TypeID != 0 && req.TypeID != p.TypeID {
		if _, err := s.typeRepo.GetByID(ctx, req.TypeID); err != nil {
			var nf dberrors.ErrNotFound
			if errors.As(err, &nf) {
				return apperrors.NewErrNotFound("property_type", req.TypeID)
			}
			s.logger.Error("update property: type repo error", "err", err)
			return apperrors.NewErrInternal("failed to validate property type")
		}
		p.TypeID = req.TypeID
	}

	// apply updates
	p.Title = strings.TrimSpace(req.Title)
	p.PropertyDescription = strings.TrimSpace(req.PropertyDescription)
	p.TransactionType = domain.TransactionType(req.TransactionType)
	p.Price = req.Price
	p.Area = req.Area
	p.PropertyAddress = strings.TrimSpace(req.PropertyAddress)
	p.City = strings.TrimSpace(req.City)
	p.PropertyStatus = domain.PropertyStatus(req.PropertyStatus)

	// geocode updated address if provided
	if p.PropertyAddress != "" {
		if lat, lon, err := s.geo.Geocode(p.PropertyAddress); err == nil {
			p.Latitude = lat
			p.Longitude = lon
		} else {
			s.logger.DebugContext(context.Background(), "geocode failed", "err", err)
		}
	}

	if err := s.repo.Update(ctx, p); err != nil {
		s.logger.Error("update property: repo update failed", "err", err)
		return apperrors.NewErrInternal("failed to update property")
	}
	s.logger.Info("update property: updated", "id", p.ID)
	return nil
}

func (s *service) List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error) {
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	list, err := s.repo.List(ctx, dr)
	if err != nil {
		s.logger.Error("list properties: repo list failed", "err", err)
		return dto.ListPropertiesResponse{}, apperrors.NewErrInternal("failed to list properties")
	}
	total := len(list)
	return dto.ListPropertiesResponse{Properties: list, Total: total}, nil
}

func (s *service) Delete(ctx context.Context, id int) (int, error) {
	deletedID, err := s.repo.Delete(ctx, id)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return 0, apperrors.NewErrNotFound("property", id)
		}
		s.logger.Error("delete property: repo delete failed", "err", err)
		return 0, apperrors.NewErrInternal("failed to delete property")
	}
	s.logger.Info("delete property: deleted", "id", deletedID)
	return deletedID, nil
}

// ToggleFavorite toggles favorite for the given user and property.
// It delegates to the favorites application service to perform existence check,
// create or delete, and maps errors accordingly.
func (s *service) ToggleFavorite(ctx context.Context, userID int, propertyID int) (bool, favdomain.Favorite, error) {
	if s.favSvc == nil {
		return false, favdomain.Favorite{}, apperrors.NewErrInternal("favorites not configured")
	}
	key := favdto.CreateFavoriteRequest{UserID: userID, PropertyID: propertyID}
	exists, err := s.favSvc.Exists(ctx, key)
	if err != nil {
		return false, favdomain.Favorite{}, err
	}
	if exists {
		// delete
		_, err := s.favSvc.Delete(ctx, key)
		if err != nil {
			return false, favdomain.Favorite{}, err
		}
		return false, favdomain.Favorite{}, nil
	}
	created, err := s.favSvc.Create(ctx, key)
	if err != nil {
		return false, favdomain.Favorite{}, err
	}
	return true, created, nil
}
