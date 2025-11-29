package propertytypeservice

import (
	"context"
	"errors"
	"strings"

	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"

	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property_type"
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

func (s *service) Create(ctx context.Context, req dto.CreatePropertyTypeRequest) (domain.PropertyType, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.PropertyType{}, apperrors.NewErrInvalidInput("name", name, "must not be empty")
	}

	pt := domain.PropertyType{Name: name}
	id, err := s.repo.Create(ctx, pt)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &ae):
			return domain.PropertyType{}, apperrors.NewErrAlreadyExists("property_type", "name", name)
		case errors.As(err, &te):
			s.logger.Error("create property type: repo timeout", "err", err)
			return domain.PropertyType{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("create property type: repo create error", "err", err)
			return domain.PropertyType{}, apperrors.NewErrInternal("failed to create property type")
		}
	}

	created, err := s.repo.GetByID(ctx, id)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("create property type: fetch created timeout", "err", err)
			return domain.PropertyType{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("create property type: failed to fetch created", "err", err)
			return domain.PropertyType{}, apperrors.NewErrInternal("failed to fetch created property type")
		}
	}
	s.logger.Info("property type created", "id", created.Id, "name", created.Name)
	return created, nil
}

func (s *service) GetByID(ctx context.Context, id int) (domain.PropertyType, error) {
	pt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return domain.PropertyType{}, apperrors.NewErrNotFound("property_type", id)
		case errors.As(err, &te):
			s.logger.Error("get by id: repo timeout", "err", err)
			return domain.PropertyType{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("get by id: repo error", "err", err)
			return domain.PropertyType{}, apperrors.NewErrInternal("failed to fetch property type")
		}
	}
	return pt, nil
}

func (s *service) Update(ctx context.Context, req dto.UpdatePropertyTypeRequest) error {
	pt, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return apperrors.NewErrNotFound("property_type", req.ID)
		case errors.As(err, &te):
			s.logger.Error("update: fetch property type timeout", "err", err)
			return apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("update: failed to fetch", "err", err)
			return apperrors.NewErrInternal("failed to fetch property type")
		}
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return apperrors.NewErrInvalidInput("name", name, "must not be empty")
	}
	pt.Name = name
	if err := s.repo.Update(ctx, pt); err != nil {
		var ae dberrors.ErrAlreadyExists
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &ae):
			return apperrors.NewErrAlreadyExists("property_type", "name", name)
		case errors.As(err, &te):
			s.logger.Error("update: repo timeout", "err", err)
			return apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("update: repo update failed", "err", err)
			return apperrors.NewErrInternal("failed to update property type")
		}
	}
	s.logger.Info("update: property type updated", "id", pt.Id)
	return nil
}

func (s *service) List(ctx context.Context, req dto.ListPropertyTypesRequest) (dto.ListPropertyTypesResponse, error) {
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	list, err := s.repo.List(ctx, dr)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("list: repo timeout", "err", err)
			return dto.ListPropertyTypesResponse{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("list: repo list failed", "err", err)
			return dto.ListPropertyTypesResponse{}, apperrors.NewErrInternal("failed to list property types")
		}
	}
	total := len(list)
	return dto.ListPropertyTypesResponse{Types: list, Total: total}, nil
}

func (s *service) Delete(ctx context.Context, id int) (int, error) {
	deletedID, err := s.repo.Delete(ctx, id)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return 0, apperrors.NewErrNotFound("property_type", id)
		case errors.As(err, &te):
			s.logger.Error("delete: repo timeout", "err", err)
			return 0, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("delete: repo delete failed", "err", err)
			return 0, apperrors.NewErrInternal("failed to delete property type")
		}
	}
	s.logger.Info("delete: property type deleted", "id", deletedID)
	return deletedID, nil
}
