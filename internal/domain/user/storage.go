package user

import "context"

type Filter struct {
	IDs      []int
	Email    string
	UserRole Role
	IsActive *bool
	Search   string
}

type ListRequest struct {
	Filter Filter
	Limit  int
	Offset int
}

type Repository interface {
	Create(ctx context.Context, user User) (int, error)
	GetByID(ctx context.Context, id int) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	Update(ctx context.Context, user User) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, req ListRequest) ([]User, int, error)
}
