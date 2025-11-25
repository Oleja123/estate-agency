package propertyservice

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
)

var _ Service = (*service)(nil)

type service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func New(repo domain.Repository, logger *slog.Logger) Service {
	return &service{repo: repo, logger: logger}
}

func (s *service) Create(ctx context.Context, req dto.CreatePropertyRequest) (domain.Property, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.Property{}, apperrors.NewErrInvalidInput("title", title, "must not be empty")
	}

	p := domain.Property{
		Title:               title,
		PropertyDescription: strings.TrimSpace(req.PropertyDescription),
		TypeID:              req.TypeID,
		TransactionType:     domain.TransactionType(req.TransactionType),
		Price:               req.Price,
		Area:                req.Area,
		PropertyAddress:     strings.TrimSpace(req.PropertyAddress),
		Latitude:            req.Latitude,
		Longitude:           req.Longitude,
		City:                strings.TrimSpace(req.City),
		CreatedBy:           req.CreatedBy,
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

	// apply updates
	p.Title = strings.TrimSpace(req.Title)
	p.PropertyDescription = strings.TrimSpace(req.PropertyDescription)
	p.TypeID = req.TypeID
	p.TransactionType = domain.TransactionType(req.TransactionType)
	p.Price = req.Price
	p.Area = req.Area
	p.PropertyAddress = strings.TrimSpace(req.PropertyAddress)
	p.Latitude = req.Latitude
	p.Longitude = req.Longitude
	p.City = strings.TrimSpace(req.City)
	p.PropertyStatus = domain.PropertyStatus(req.PropertyStatus)

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

func (s *service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("property", id)
		}
		s.logger.Error("delete property: repo delete failed", "err", err)
		return apperrors.NewErrInternal("failed to delete property")
	}
	s.logger.Info("delete property: deleted", "id", id)
	return nil
}
