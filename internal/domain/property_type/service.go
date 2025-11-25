package propertytypeservice

import "context"

// Service defines business operations for property types.
// This interface belongs to the domain layer and expresses core business
// use-cases without depending on infrastructure.
type Service interface {
	Create(ctx context.Context, name string) (int, error)
	GetByID(ctx context.Context, id int) (PropertyType, error)
	GetByName(ctx context.Context, name string) (PropertyType, error)
	Update(ctx context.Context, pt PropertyType) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, req ListRequest) ([]PropertyType, error)
}
