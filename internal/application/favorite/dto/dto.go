package dto

import (
	"github.com/Oleja123/estate-agency/internal/domain/favorite"
)

// CreateFavoriteRequest is the compact DTO that contains user and property IDs.
type CreateFavoriteRequest struct {
	UserID     int `json:"user_id"`
	PropertyID int `json:"property_id"`
}

// ListFavoritesRequest is used to list favorites with optional filters and pagination.
type ListFavoritesRequest struct {
	Filter favorite.Filter `json:"filter"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// ListFavoritesResponse is returned when listing favorites.
type ListFavoritesResponse struct {
	Favorites []favorite.Favorite `json:"favorites"`
	Total     int                 `json:"total"`
}
