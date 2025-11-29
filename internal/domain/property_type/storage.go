package propertytypeservice

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
	Delete(ctx context.Context, id int) (int, error)
	// List returns matching property types and the total count of matching rows (ignoring limit/offset)
	List(ctx context.Context, req ListRequest) ([]PropertyType, int, error)
}
