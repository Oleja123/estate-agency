package property

import "context"

type Filter struct {
	IDs             []int
	TypeIDs         []int
	TransactionType TransactionType
	City            string
	PropertyStatus  PropertyStatus
	CreatedBy       int
	MinPrice        float64
	MaxPrice        float64
	MinArea         float64
	MaxArea         float64
	Search          string
	Latitude        float64
	Longitude       float64
	RadiusKm        float64
}

type ListRequest struct {
	Filter Filter
	Limit  int
	Offset int
}

type Repository interface {
	Create(ctx context.Context, property Property) (int, error)
	GetByID(ctx context.Context, id int) (Property, error)
	Update(ctx context.Context, property Property) error
	Delete(ctx context.Context, id int) (int, error)
	List(ctx context.Context, req ListRequest) ([]Property, error)
}
