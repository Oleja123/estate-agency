package propertytypeservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property_type"
)

type Service interface {
	Create(ctx context.Context, req dto.CreatePropertyTypeRequest) (domain.PropertyType, error)
	GetByID(ctx context.Context, id int) (domain.PropertyType, error)
	Update(ctx context.Context, req dto.UpdatePropertyTypeRequest) (domain.PropertyType, error)
	List(ctx context.Context, req dto.ListPropertyTypesRequest) (dto.ListPropertyTypesResponse, error)
	Delete(ctx context.Context, id int) (int, error)
}
