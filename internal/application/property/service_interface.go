package propertyservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
)

type Service interface {
	Create(ctx context.Context, req dto.CreatePropertyRequest) (domain.Property, error)
	GetByID(ctx context.Context, id int) (domain.Property, error)
	Update(ctx context.Context, req dto.UpdatePropertyRequest) error
	List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error)
	Delete(ctx context.Context, id int) error
}
