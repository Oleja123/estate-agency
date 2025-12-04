package dto

import (
	"github.com/Oleja123/estate-agency/internal/application/property/dto"
	"github.com/Oleja123/estate-agency/internal/domain/favorite"
)

type CreateFavoriteRequest struct {
	UserID     int `json:"user_id"`
	PropertyID int `json:"property_id"`
}

type ListFavoritesRequest struct {
	Filter favorite.Filter `json:"filter"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type ListFavoritesResponse struct {
	Favorites []dto.PropertyDTO `json:"favorites"`
	Total     int               `json:"total"`
}
