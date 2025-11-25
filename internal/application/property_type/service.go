package propertytypesvc

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	apperrors "github.com/Oleja123/estate-agency/internal/application/property_type/errors"
	propertytype "github.com/Oleja123/estate-agency/internal/domain/property_type"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
)

// Service is the application-level service API for property types. It accepts
// application DTOs and returns application-level errors.
type Service interface {
	Create(ctx context.Context, req dto.CreateRequest) (int, error)
	GetByID(ctx context.Context, id int) (propertytype.PropertyType, error)
	GetByName(ctx context.Context, name string) (propertytype.PropertyType, error)
	Update(ctx context.Context, req dto.UpdateRequest) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, req propertytype.ListRequest) ([]propertytype.PropertyType, error)
}

type service struct {
	repo   propertytype.Repository
	logger *slog.Logger
}

// New returns a new application Service backed by the provided repository.
func New(repo propertytype.Repository, logger *slog.Logger) Service {
	return &service{repo: repo, logger: logger}
}

func (s *service) Create(ctx context.Context, req dto.CreateRequest) (int, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, apperrors.NewErrInvalidInput("name", name, "must not be empty")
	}

	// ensure uniqueness
	if existing, err := s.repo.GetByName(ctx, name); err == nil && existing.Id != 0 {
		return 0, apperrors.NewErrAlreadyExists("property_type", "property_name", name)
	} else if err != nil {
		// If it's a not found error from repository, continue; otherwise propagate
		if _, ok := err.(dberrors.ErrNotFound); !ok {
			return 0, fmt.Errorf("failed to check existing property type: %w", err)
		}
	}

	id, err := s.repo.Create(ctx, propertytype.PropertyType{Name: name})
	if err != nil {
		return 0, fmt.Errorf("failed to create property type: %w", err)
	}
	return id, nil
}

func (s *service) GetByID(ctx context.Context, id int) (propertytype.PropertyType, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetByName(ctx context.Context, name string) (propertytype.PropertyType, error) {
	return s.repo.GetByName(ctx, name)
}

func (s *service) Update(ctx context.Context, req dto.UpdateRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return apperrors.NewErrInvalidInput("name", name, "must not be empty")
	}

	// ensure unique name (if another record exists with same name)
	if existing, err := s.repo.GetByName(ctx, name); err == nil && existing.Id != 0 && existing.Id != req.Id {
		return apperrors.NewErrAlreadyExists("property_type", "property_name", name)
	} else if err != nil {
		// if error is not not-found from repository, propagate
		if _, ok := err.(dberrors.ErrNotFound); !ok {
			return fmt.Errorf("failed to check existing property type: %w", err)
		}
	}

	pt := propertytype.PropertyType{Id: req.Id, Name: name}
	if err := s.repo.Update(ctx, pt); err != nil {
		return fmt.Errorf("failed to update property type: %w", err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete property type: %w", err)
	}
	return nil
}

func (s *service) List(ctx context.Context, req propertytype.ListRequest) ([]propertytype.PropertyType, error) {
	return s.repo.List(ctx, req)
}
