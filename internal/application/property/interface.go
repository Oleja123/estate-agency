package propertyservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	favdomain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
)

type Service interface {
	Create(ctx context.Context, userID int, req dto.CreatePropertyRequest) (domain.Property, error)
	GetByID(ctx context.Context, id int) (domain.Property, error)
	Update(ctx context.Context, req dto.UpdatePropertyRequest) error
	List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error)
	Delete(ctx context.Context, id int) (int, error)
	// ToggleFavorite toggles a favorite for the given user and property.
	// Returns (created, favorite, error). If created==true, favorite is the created favorite.
	// If created==false, the favorite was deleted and favorite is zero value.
	ToggleFavorite(ctx context.Context, userID int, propertyID int) (bool, favdomain.Favorite, error)
}
