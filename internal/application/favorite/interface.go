package favoriteservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	prop "github.com/Oleja123/estate-agency/internal/domain/property"
)

type Service interface {
	Create(ctx context.Context, req dto.CreateFavoriteRequest) (int, error)
	GetByUserAndProperty(ctx context.Context, key dto.CreateFavoriteRequest) (prop.Property, error)
	Delete(ctx context.Context, key dto.CreateFavoriteRequest) (int, error)
	List(ctx context.Context, req dto.ListFavoritesRequest) (dto.ListFavoritesResponse, error)
	Exists(ctx context.Context, key dto.CreateFavoriteRequest) (bool, error)
}
