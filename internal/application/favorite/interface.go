package favoriteservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/favorite"
)

type Service interface {
	Create(ctx context.Context, req dto.CreateFavoriteRequest) (domain.Favorite, error)
	GetByUserAndProperty(ctx context.Context, key dto.CreateFavoriteRequest) (domain.Favorite, error)
	Delete(ctx context.Context, key dto.CreateFavoriteRequest) (int, error)
	List(ctx context.Context, req dto.ListFavoritesRequest) (dto.ListFavoritesResponse, error)
	Exists(ctx context.Context, key dto.CreateFavoriteRequest) (bool, error)
}
