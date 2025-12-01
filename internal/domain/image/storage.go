package image

import "context"

type ImageRepository interface {
	Create(ctx context.Context, img PropertyImage) (int, error)

	CreateMany(ctx context.Context, imgs []PropertyImage) ([]int, error)

	GetByID(ctx context.Context, id int) (PropertyImage, error)

	ListByProperty(ctx context.Context, propertyID int) ([]PropertyImage, error)

	Delete(ctx context.Context, id int) (int, error)

	DeleteMany(ctx context.Context, propertyID int) ([]int, error)
}
