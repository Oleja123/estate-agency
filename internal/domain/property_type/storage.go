package propertytype

import "context"

type Filter struct {
	IDs    []int
	Name   string
	Search string
}

type ListRequest struct {
	Filter Filter
	Limit  int
	Offset int
}

type Repository interface {
	Create(ctx context.Context, propertyType PropertyType) (int, error)
	GetByID(ctx context.Context, id int) (PropertyType, error)
	GetByName(ctx context.Context, name string) (PropertyType, error)
	Update(ctx context.Context, propertyType PropertyType) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, req ListRequest) ([]ListRequest, error)
}
