package favorite

import "context"

type Filter struct {
	UserID      int
	PropertyID  int
	UserIDs     []int
	PropertyIDs []int
}

type ListRequest struct {
	Filter Filter
	Limit  int
	Offset int
}

type Repository interface {
	Create(ctx context.Context, favorite Favorite) error
	GetByUserAndProperty(ctx context.Context, userID, propertyID int) (Favorite, error)
	Delete(ctx context.Context, userID, propertyID int) (int, error)

	List(ctx context.Context, req ListRequest) ([]Favorite, int, error)
	Exists(ctx context.Context, userID, propertyID int) (bool, error)
}
