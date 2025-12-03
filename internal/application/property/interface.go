package propertyservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
)

type Service interface {
	Create(ctx context.Context, userID int, req dto.CreatePropertyRequest) (dto.PropertyDTO, error)
	GetByID(ctx context.Context, id int) (dto.PropertyDTO, error)
	// GetByIDWithFavorite returns property with favorite flag set (userID may be 0)
	GetByIDWithFavorite(ctx context.Context, id int, userID int) (dto.PropertyDTO, error)
	Update(ctx context.Context, req dto.UpdatePropertyRequest) (dto.PropertyDTO, error)
	List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error)
	Delete(ctx context.Context, id int) (int, error)
	ToggleFavorite(ctx context.Context, userID int, propertyID int) (bool, dto.PropertyDTO, error)
}
