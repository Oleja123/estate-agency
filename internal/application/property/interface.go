package propertyservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	prop "github.com/Oleja123/estate-agency/internal/domain/property"
)

type Service interface {
	Create(ctx context.Context, userID int, req dto.CreatePropertyRequest) (prop.Property, error)
	GetByID(ctx context.Context, id int) (prop.Property, error)
	// GetByIDWithFavorite returns property and whether the given user has it in favorites (userID may be 0)
	GetByIDWithFavorite(ctx context.Context, id int, userID int) (prop.Property, bool, error)
	Update(ctx context.Context, req dto.UpdatePropertyRequest) (prop.Property, error)
	List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error)
	Delete(ctx context.Context, id int) (int, error)
	ToggleFavorite(ctx context.Context, userID int, propertyID int) (bool, prop.Property, error)
}
