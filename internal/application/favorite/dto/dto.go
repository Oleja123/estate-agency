package dto

import (
	"github.com/Oleja123/estate-agency/internal/domain/favorite"
	prop "github.com/Oleja123/estate-agency/internal/domain/property"
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
	Favorites []prop.Property `json:"favorites"`
	Total     int             `json:"total"`
}
